package scanner

import (
	"context"
	"fmt"
	"gearr/helper"
	"gearr/model"
	"gearr/server/repository"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

var defaultFileExtensions = []string{".mkv", ".mp4", ".avi", ".mov", ".wmv", ".flv", ".webm", ".m4v"}

type Scanner struct {
	config       model.ScannerConfig
	repo         repository.Repository
	scanChan     chan struct{}
	statusChan   chan *model.ScannerNotification
	mu           sync.RWMutex
	currentScan  *model.LibraryScan
	nextScanTime *time.Time
	scheduler    Scheduler
}

type Scheduler interface {
	ScheduleJobRequest(ctx context.Context, jobRequest *model.JobRequest) (*model.Job, error)
}

func NewScanner(config model.ScannerConfig, repo repository.Repository, scheduler Scheduler) *Scanner {
	if len(config.FileExtensions) == 0 {
		config.FileExtensions = defaultFileExtensions
	}
	return &Scanner{
		config:     config,
		repo:       repo,
		scanChan:   make(chan struct{}, 1),
		statusChan: make(chan *model.ScannerNotification, 100),
		scheduler:  scheduler,
	}
}

func (s *Scanner) Run(wg *sync.WaitGroup, ctx context.Context) {
	defer wg.Done()
	log.Info("starting library scanner")

	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()

	s.mu.Lock()
	nextScan := time.Now().Add(s.config.Interval)
	s.nextScanTime = &nextScan
	s.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			log.Info("stopping library scanner")
			return
		case <-ticker.C:
			if s.config.Enabled {
				s.performScan(ctx)
			}
			s.mu.Lock()
			nextScan := time.Now().Add(s.config.Interval)
			s.nextScanTime = &nextScan
			s.mu.Unlock()
		case <-s.scanChan:
			s.performScan(ctx)
		}
	}
}

func (s *Scanner) TriggerScan() {
	select {
	case s.scanChan <- struct{}{}:
	default:
		log.Debug("scan already queued")
	}
}

func (s *Scanner) GetStatus() *model.ScannerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := &model.ScannerStatus{
		Enabled:    s.config.Enabled,
		IsScanning: s.currentScan != nil && s.currentScan.Status == model.ScanRunning,
		Config:     s.config,
		NextScanAt: s.nextScanTime,
	}

	if s.currentScan != nil {
		status.LastScan = s.currentScan
	}

	return status
}

func (s *Scanner) GetNotificationChan() <-chan *model.ScannerNotification {
	return s.statusChan
}

func (s *Scanner) GetHistory(ctx context.Context, limit int) ([]*model.LibraryScan, error) {
	return s.repo.GetScanHistory(ctx, limit)
}

func (s *Scanner) performScan(ctx context.Context) {
	s.mu.Lock()
	if s.currentScan != nil && s.currentScan.Status == model.ScanRunning {
		s.mu.Unlock()
		log.Warn("scan already in progress")
		return
	}

	scan := &model.LibraryScan{
		Id:        uuid.New().String(),
		StartedAt: time.Now(),
		Status:    model.ScanRunning,
	}
	s.currentScan = scan
	s.mu.Unlock()

	s.sendNotification(&model.ScannerNotification{
		Type:   "scan_started",
		ScanId: scan.Id,
		Status: model.ScanRunning,
	})

	err := s.repo.CreateScan(ctx, scan)
	if err != nil {
		log.Errorf("failed to create scan record: %v", err)
		s.completeScan(scan, model.ScanFailed, fmt.Sprintf("failed to create scan record: %v", err))
		return
	}

	for _, path := range s.config.Paths {
		if ctx.Err() != nil {
			s.completeScan(scan, model.ScanFailed, "scan cancelled")
			return
		}

		s.scanDirectory(ctx, scan, path)
	}

	s.completeScan(scan, model.ScanCompleted, "")
}

func (s *Scanner) scanDirectory(ctx context.Context, scan *model.LibraryScan, rootPath string) {
	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Warnf("error accessing path %s: %v", path, err)
			return nil
		}

		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !s.isValidExtension(ext) {
			return nil
		}

		s.mu.Lock()
		scan.FilesFound++
		currentFiles := scan.FilesFound
		s.mu.Unlock()

		s.sendNotification(&model.ScannerNotification{
			Type:        "scan_progress",
			ScanId:      scan.Id,
			Progress:    currentFiles,
			FilesFound:  scan.FilesFound,
			CurrentPath: path,
			Status:      model.ScanRunning,
		})

		fileInfo, err := os.Stat(path)
		if err != nil {
			log.Warnf("cannot stat file %s: %v", path, err)
			return nil
		}

		if fileInfo.Size() < s.config.MinFileSize {
			s.mu.Lock()
			scan.FilesSkippedSize++
			s.mu.Unlock()
			log.Debugf("skipping file %s: size %d below threshold %d", path, fileInfo.Size(), s.config.MinFileSize)
			return nil
		}

		existingFile, err := s.repo.GetScannedFile(ctx, path)
		if err != nil && err != repository.ErrElementNotFound {
			log.Warnf("error checking scanned file %s: %v", path, err)
			return nil
		}

		if existingFile != nil && existingFile.Queued {
			s.mu.Lock()
			scan.FilesSkippedExist++
			s.mu.Unlock()
			log.Debugf("skipping already queued file: %s", path)
			return nil
		}

		existingJob, err := s.repo.GetJobByPath(ctx, path)
		if err != nil && err != repository.ErrElementNotFound {
			log.Warnf("error checking existing job for %s: %v", path, err)
			return nil
		}
		if existingJob != nil {
			s.mu.Lock()
			scan.FilesSkippedExist++
			s.mu.Unlock()
			log.Debugf("skipping file with existing job: %s", path)
			return nil
		}

		codec, err := helper.DetectCodec(path)
		if err != nil {
			log.Warnf("failed to detect codec for %s: %v", path, err)
			return nil
		}

		if codec == "hevc" || codec == "x265" {
			s.mu.Lock()
			scan.FilesSkippedCodec++
			s.mu.Unlock()
			log.Debugf("skipping already x265/hevc file: %s", path)
			return nil
		}

		scannedFile := &model.ScannedFile{
			Id:            uuid.New().String(),
			FilePath:      path,
			FileSize:      fileInfo.Size(),
			Codec:         codec,
			LastScannedAt: time.Now(),
			Queued:        false,
			ScanId:        scan.Id,
		}

		err = s.repo.UpsertScannedFile(ctx, scannedFile)
		if err != nil {
			log.Warnf("failed to upsert scanned file %s: %v", path, err)
			return nil
		}

		jobRequest := &model.JobRequest{
			SourcePath: path,
		}

		_, err = s.scheduler.ScheduleJobRequest(ctx, jobRequest)
		if err != nil {
			log.Warnf("failed to queue job for %s: %v", path, err)
			return nil
		}

		scannedFile.Queued = true
		err = s.repo.UpsertScannedFile(ctx, scannedFile)
		if err != nil {
			log.Warnf("failed to update scanned file status %s: %v", path, err)
		}

		s.mu.Lock()
		scan.FilesQueued++
		s.mu.Unlock()

		log.Infof("queued file for transcoding: %s (codec: %s, size: %d)", path, codec, fileInfo.Size())
		return nil
	})

	if err != nil {
		log.Errorf("error walking directory %s: %v", rootPath, err)
	}
}

func (s *Scanner) isValidExtension(ext string) bool {
	for _, validExt := range s.config.FileExtensions {
		if strings.EqualFold(ext, validExt) {
			return true
		}
	}
	return false
}

func (s *Scanner) completeScan(scan *model.LibraryScan, status model.ScanStatus, errorMsg string) {
	s.mu.Lock()
	now := time.Now()
	scan.CompletedAt = &now
	scan.Status = status
	scan.ErrorMessage = errorMsg
	s.mu.Unlock()

	err := s.repo.UpdateScan(context.Background(), scan)
	if err != nil {
		log.Errorf("failed to update scan record: %v", err)
	}

	s.sendNotification(&model.ScannerNotification{
		Type:         "scan_completed",
		ScanId:       scan.Id,
		FilesFound:   scan.FilesFound,
		FilesQueued:  scan.FilesQueued,
		FilesSkipped: scan.FilesSkippedSize + scan.FilesSkippedCodec + scan.FilesSkippedExist,
		Status:       status,
		ErrorMessage: errorMsg,
	})

	log.Infof("scan completed: %d files found, %d queued, %d skipped (size: %d, codec: %d, exists: %d)",
		scan.FilesFound, scan.FilesQueued,
		scan.FilesSkippedSize+scan.FilesSkippedCodec+scan.FilesSkippedExist,
		scan.FilesSkippedSize, scan.FilesSkippedCodec, scan.FilesSkippedExist)
}

func (s *Scanner) sendNotification(notification *model.ScannerNotification) {
	select {
	case s.statusChan <- notification:
	default:
		log.Warn("notification channel full, dropping notification")
	}
}
