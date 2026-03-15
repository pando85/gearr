package watcher

import (
	"context"
	"fmt"
	"gearr/helper"
	"gearr/helper/codec"
	"gearr/model"
	"gearr/server/repository"
	"gearr/server/scheduler"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type WatcherStatus struct {
	Active            bool      `json:"active"`
	WatchedPaths      []string  `json:"watched_paths"`
	FilesDetected     int       `json:"files_detected"`
	FilesQueued       int       `json:"files_queued"`
	LastDetectionTime time.Time `json:"last_detection_time"`
}

type Config struct {
	Paths        []string      `mapstructure:"paths"`
	Enabled      bool          `mapstructure:"enabled"`
	DebounceTime time.Duration `mapstructure:"debounceTime"`
	FilePatterns []string      `mapstructure:"patterns"`
	MinFileSize  int64         `mapstructure:"minFileSize"`
	DownloadPath string        `mapstructure:"downloadPath"`
}

type Watcher struct {
	config          Config
	scheduler       scheduler.Scheduler
	repo            repository.Repository
	fsnotifyWatcher *fsnotify.Watcher
	status          WatcherStatus
	statusMutex     sync.RWMutex
	debounceMap     map[string]*time.Timer
	debounceMutex   sync.Mutex
	ctx             context.Context
	cancel          context.CancelFunc
}

func NewWatcher(config Config, sched scheduler.Scheduler, repo repository.Repository) (*Watcher, error) {
	if !config.Enabled || len(config.Paths) == 0 {
		log.Info("folder watcher is disabled or no paths configured")
		return &Watcher{config: config, scheduler: sched, repo: repo}, nil
	}

	return &Watcher{
		config:      config,
		scheduler:   sched,
		repo:        repo,
		debounceMap: make(map[string]*time.Timer),
		status: WatcherStatus{
			Active:       false,
			WatchedPaths: config.Paths,
		},
	}, nil
}

func (w *Watcher) Run(wg *sync.WaitGroup, ctx context.Context) {
	if !w.config.Enabled || len(w.config.Paths) == 0 {
		return
	}

	w.ctx, w.cancel = context.WithCancel(ctx)

	fsnWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Errorf("failed to create fsnotify watcher: %v", err)
		return
	}
	w.fsnotifyWatcher = fsnWatcher

	for _, path := range w.config.Paths {
		if err := w.AddPath(path); err != nil {
			log.Errorf("failed to add watch path %s: %v", path, err)
		}
	}

	w.setStatusActive(true)
	log.Infof("folder watcher started, monitoring %d paths", len(w.config.Paths))

	wg.Add(1)
	go func() {
		defer wg.Done()
		w.watch(ctx)
	}()
}

func (w *Watcher) watch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			w.stop()
			return
		case event, ok := <-w.fsnotifyWatcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Rename == fsnotify.Rename {
				w.handleFileEvent(event.Name)
			}
		case err, ok := <-w.fsnotifyWatcher.Errors:
			if !ok {
				return
			}
			log.Errorf("fsnotify error: %v", err)
		}
	}
}

func (w *Watcher) stop() {
	w.setStatusActive(false)
	if w.fsnotifyWatcher != nil {
		w.fsnotifyWatcher.Close()
	}
	w.debounceMutex.Lock()
	for _, timer := range w.debounceMap {
		timer.Stop()
	}
	w.debounceMap = make(map[string]*time.Timer)
	w.debounceMutex.Unlock()
	log.Info("folder watcher stopped")
}

func (w *Watcher) handleFileEvent(filePath string) {
	w.debounceMutex.Lock()
	defer w.debounceMutex.Unlock()

	if timer, exists := w.debounceMap[filePath]; exists {
		timer.Stop()
	}

	w.debounceMap[filePath] = time.AfterFunc(w.config.DebounceTime, func() {
		w.processFile(filePath)
		w.debounceMutex.Lock()
		delete(w.debounceMap, filePath)
		w.debounceMutex.Unlock()
	})
}

func (w *Watcher) processFile(filePath string) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Errorf("failed to stat file %s: %v", filePath, err)
		return
	}

	if fileInfo.IsDir() {
		return
	}

	extension := filepath.Ext(filePath)
	if len(extension) > 0 {
		extension = extension[1:]
	}
	if !helper.ValidExtension(extension) {
		return
	}

	if !w.matchesPattern(filepath.Base(filePath)) {
		return
	}

	if fileInfo.Size() < w.config.MinFileSize {
		return
	}

	relativePath, err := filepath.Rel(w.config.DownloadPath, filePath)
	if err != nil {
		log.Errorf("failed to get relative path for %s: %v", filePath, err)
		relativePath = filePath
	}

	existingJob, err := w.repo.GetJobByPath(w.ctx, relativePath)
	if err != nil {
		log.Errorf("failed to check existing job for %s: %v", relativePath, err)
		return
	}
	if existingJob != nil {
		log.Debugf("file %s already has a job, skipping", relativePath)
		return
	}

	existingDetection, err := w.repo.GetFileProcessingByPath(w.ctx, relativePath)
	if err != nil {
		log.Errorf("failed to check existing detection for %s: %v", relativePath, err)
		return
	}

	detectedAt := time.Now()
	fp := &model.FileProcessing{
		Path:       relativePath,
		DetectedAt: detectedAt,
		Source:     model.WatcherSource,
	}

	if !codec.NeedsTranscoding(relativePath) {
		fp.Status = model.X265Status
		fp.Message = "file already encoded in x265 or compatible codec"
		if err := w.repo.AddFileProcessing(w.ctx, fp); err != nil {
			log.Errorf("failed to record file processing for %s: %v", relativePath, err)
		}
		log.Infof("watcher: file %s is already x265, skipping", relativePath)
		return
	}

	if existingDetection != nil && existingDetection.Status == model.QueuedStatus {
		log.Debugf("file %s already queued, skipping", relativePath)
		return
	}

	jobRequest := &model.JobRequest{
		SourcePath:      relativePath,
		DestinationPath: "",
	}

	job, err := w.scheduler.ScheduleJobRequest(w.ctx, jobRequest)
	if err != nil {
		fp.Status = model.ErrorStatus
		fp.Message = fmt.Sprintf("failed to queue job: %v", err)
		if err := w.repo.AddFileProcessing(w.ctx, fp); err != nil {
			log.Errorf("failed to record file processing error for %s: %v", relativePath, err)
		}
		log.Errorf("failed to queue job for %s: %v", relativePath, err)
		return
	}

	fp.Status = model.QueuedStatus
	fp.Message = "file queued for transcoding"
	fp.JobId = &job.Id
	if err := w.repo.AddFileProcessing(w.ctx, fp); err != nil {
		log.Errorf("failed to record file processing for %s: %v", relativePath, err)
	}

	w.statusMutex.Lock()
	w.status.FilesDetected++
	w.status.FilesQueued++
	w.status.LastDetectionTime = detectedAt
	w.statusMutex.Unlock()

	log.Infof("watcher: detected and queued %s for transcoding (job: %s)", relativePath, job.Id)
}

func (w *Watcher) matchesPattern(filename string) bool {
	if len(w.config.FilePatterns) == 0 {
		return true
	}

	for _, pattern := range w.config.FilePatterns {
		matched, err := filepath.Match(pattern, filename)
		if err != nil {
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

func (w *Watcher) AddPath(path string) error {
	if w.fsnotifyWatcher == nil {
		return fmt.Errorf("watcher not initialized")
	}

	if err := w.fsnotifyWatcher.Add(path); err != nil {
		return err
	}

	w.statusMutex.Lock()
	found := false
	for _, p := range w.status.WatchedPaths {
		if p == path {
			found = true
			break
		}
	}
	if !found {
		w.status.WatchedPaths = append(w.status.WatchedPaths, path)
	}
	w.statusMutex.Unlock()

	log.Infof("watcher: added path %s", path)
	return nil
}

func (w *Watcher) RemovePath(path string) error {
	if w.fsnotifyWatcher == nil {
		return fmt.Errorf("watcher not initialized")
	}

	if err := w.fsnotifyWatcher.Remove(path); err != nil {
		return err
	}

	w.statusMutex.Lock()
	var newPaths []string
	for _, p := range w.status.WatchedPaths {
		if p != path {
			newPaths = append(newPaths, p)
		}
	}
	w.status.WatchedPaths = newPaths
	w.statusMutex.Unlock()

	log.Infof("watcher: removed path %s", path)
	return nil
}

func (w *Watcher) GetStatus() WatcherStatus {
	w.statusMutex.RLock()
	defer w.statusMutex.RUnlock()
	return w.status
}

func (w *Watcher) setStatusActive(active bool) {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	w.status.Active = active
}

func (w *Watcher) GetRecentDetections(limit int) ([]*model.FileProcessing, error) {
	if w.repo == nil {
		return nil, fmt.Errorf("repository not available")
	}
	return w.repo.GetRecentFileProcessings(w.ctx, limit, model.WatcherSource)
}

func (w *Watcher) IsEnabled() bool {
	return w.config.Enabled
}

func (w *Watcher) ParseWatchPaths(pathsStr string) []string {
	if pathsStr == "" {
		return nil
	}
	var paths []string
	for _, p := range strings.Split(pathsStr, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			paths = append(paths, p)
		}
	}
	return paths
}

func GenerateTargetPath(sourcePath string) string {
	return codec.FormatTargetName(sourcePath)
}

func NewUUID() uuid.UUID {
	id, _ := uuid.NewUUID()
	return id
}
