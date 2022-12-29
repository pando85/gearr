package task

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/avast/retry-go"
	"gopkg.in/vansante/go-ffprobe.v2"
	"hash"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"transcoder/helper"
	"transcoder/helper/command"
	"transcoder/helper/progress"
	"transcoder/model"
)

const RESET_LINE = "\r\033[K"
const DOWNLOAD_QUEUE_SIZE = 1

var ffmpegSpeedRegex = regexp.MustCompile(`speed=(\d*\.?\d+)x`)
var ErrorJobNotFound = errors.New("job Not found")

type FFMPEGProgress struct {
	duration int
	speed    float64
	percent  float64
}
type EncodeWorker struct {
	model.Manager
	name          string
	ctx           context.Context
	cancelContext context.CancelFunc

	downloadChan chan *model.WorkTaskEncode
	encodeChan   chan *model.WorkTaskEncode
	uploadChan   chan *model.WorkTaskEncode
	workerConfig Config
	tempPath     string
	mu           sync.RWMutex
	terminal     *ConsoleWorkerPrinter
}

func NewEncodeWorker(ctx context.Context, workerConfig Config, workerName string, printer *ConsoleWorkerPrinter) *EncodeWorker {
	newCtx, cancel := context.WithCancel(ctx)
	tempPath := filepath.Join(workerConfig.TemporalPath, fmt.Sprintf("worker-%s", workerName))
	encodeWorker := &EncodeWorker{
		name:          workerName,
		ctx:           newCtx,
		cancelContext: cancel,
		workerConfig:  workerConfig,
		downloadChan:  make(chan *model.WorkTaskEncode, 100),
		encodeChan:    make(chan *model.WorkTaskEncode, 100),
		uploadChan:    make(chan *model.WorkTaskEncode, 100),
		tempPath:      tempPath,
		terminal:      printer,
	}

	return encodeWorker
}

func (E *EncodeWorker) Initialize() {
	E.resumeJobs()
	go E.terminal.Render()
	go E.downloadQueue()
	go E.uploadQueue()
	go E.encodeQueue()

}

func (E *EncodeWorker) resumeJobs() {
	err := filepath.Walk(E.tempPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".json" {
			filepath.Base(path)
			taskEncode := E.readTaskStatusFromDiskByPath(path)

			if taskEncode.LastState.IsDownloading() {
				E.downloadChan <- taskEncode.Task
				return nil
			}
			if taskEncode.LastState.IsEncoding() {
				E.encodeChan <- taskEncode.Task
				return nil
			}
			if taskEncode.LastState.IsUploading() {
				E.uploadChan <- taskEncode.Task
				return nil
			}
		}

		return nil
	})

	if err != nil {
		panic(err)
	}
}
func durToSec(dur string) (sec int) {
	durAry := strings.Split(dur, ":")
	if len(durAry) != 3 {
		return
	}
	hr, _ := strconv.Atoi(durAry[0])
	sec = hr * (60 * 60)
	min, _ := strconv.Atoi(durAry[1])
	sec += min * (60)
	second, _ := strconv.Atoi(durAry[2])
	sec += second
	return
}
func getSpeed(res string) float64 {
	rs := ffmpegSpeedRegex.FindStringSubmatch(res)
	if len(rs) == 0 {
		return -1
	}
	speed, err := strconv.ParseFloat(rs[1], 64)
	if err != nil {
		return -1
	}
	return speed

}

func getDuration(res string) int {
	i := strings.Index(res, "time=")
	if i >= 0 {
		time := res[i+5:]
		if len(time) > 8 {
			time = time[0:8]
			sec := durToSec(time)
			return sec
		}
	}
	return -1
}

/*func printProgress(ctx context.Context, reader *progress.Reader, size int64, wg *sync.WaitGroup, label string) {
	wg.Add(1)
	//TODO no calcula be el temps/velocitat, donat que el calcula desde el principi i hauria de calcular amb els ultims X segons
	progressChan := progress.NewTicker(ctx, reader, size, 1*time.Second)

	for p := range progressChan {
		blocks := int((p.Percent() / 100) * 50)
		line := "|<"
		for i := 0; i < blocks; i++ {
			line = line + "#"
		}
		for i := 0; i < (50 - blocks); i++ {
			line = line + "-"
		}
		line = line + ">|"
		fmt.Printf("%s%s %s Speed:%s/s Remaining:%s EstimatedAt: %02d:%02d", RESET_LINE, label, line, bytesize.New(p.Speed()), durafmt.Parse(p.Remaining()).LimitFirstN(2).String(), p.Estimated().Hour(), p.Estimated().Minute())
	}
	fmt.Printf("\n")
	wg.Done()

}*/

func (J *EncodeWorker) IsTypeAccepted(jobType string) bool {
	return jobType == string(model.EncodeJobType)
}

func (J *EncodeWorker) AcceptJobs() bool {
	now := time.Now()
	if J.workerConfig.Paused {
		return false
	}
	if J.workerConfig.HaveSettedPeriodTime() {
		startAfter := time.Date(now.Year(), now.Month(), now.Day(), J.workerConfig.StartAfter.Hour, J.workerConfig.StartAfter.Minute, 0, 0, now.Location())
		stopAfter := time.Date(now.Year(), now.Month(), now.Day(), J.workerConfig.StopAfter.Hour, J.workerConfig.StopAfter.Minute, 0, 0, now.Location())
		return now.After(startAfter) && now.Before(stopAfter)
	}
	return !J.isQueueFull()
}

func (j *EncodeWorker) dowloadFile(job *model.WorkTaskEncode, track *TaskTracks) (err error) {
	err = retry.Do(func() error {
		track.UpdateValue(0)
		resp, err := http.Get(job.TaskEncode.DownloadURL)
		if err != nil {
			return err
		}
		if resp.StatusCode == http.StatusNotFound {
			return ErrorJobNotFound
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf(fmt.Sprintf("not 200 respose in dowload code %d", resp.StatusCode))
		}
		defer resp.Body.Close()
		size, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
		track.SetTotal(size)
		if err != nil {
			return err
		}
		_, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
		if err != nil {
			return err
		}

		job.SourceFilePath = filepath.Join(job.WorkDir, fmt.Sprintf("%s%s", job.TaskEncode.Id.String(), filepath.Ext(params["filename"])))
		dowloadFile, err := os.Create(job.SourceFilePath)
		if err != nil {
			return err
		}

		defer dowloadFile.Close()

		reader := NewProgressTrackStream(track, resp.Body)

		_, err = io.Copy(dowloadFile, reader)

		sha256String := hex.EncodeToString(reader.SumSha())
		bodyString := ""

		err = retry.Do(func() error {
			respsha256, err := http.Get(job.TaskEncode.ChecksumURL)
			defer respsha256.Body.Close()
			if err != nil {
				return err
			}
			if respsha256.StatusCode != http.StatusOK {
				return fmt.Errorf(fmt.Sprintf("not 200 respose in sha265 code %d", respsha256.StatusCode))
			}

			bodyBytes, err := ioutil.ReadAll(respsha256.Body)
			if err != nil {
				return err
			}
			bodyString = string(bodyBytes)
			return nil
		}, retry.Delay(time.Second*5),
			retry.Attempts(10),
			retry.LastErrorOnly(true),
			retry.OnRetry(func(n uint, err error) {
				j.terminal.Log("Error %d on calculate checksum of downloaded job %s", err.Error())
			}),
			retry.RetryIf(func(err error) bool {
				return !errors.Is(err, context.Canceled)
			}))
		if err != nil {
			return err
		}

		if sha256String != bodyString {
			return fmt.Errorf("Checksum error on download source:%s downloaded:%s", bodyString, sha256String)
		}

		track.UpdateValue(size)
		return nil
	}, retry.Delay(time.Second*5),
		retry.DelayType(retry.FixedDelay),
		retry.Attempts(180), //15 min
		retry.LastErrorOnly(true),
		retry.OnRetry(func(n uint, err error) {
			j.terminal.Log("Error on downloading job %s", err.Error())
		}),
		retry.RetryIf(func(err error) bool {
			return !(errors.Is(err, context.Canceled) || errors.Is(err, ErrorJobNotFound))
		}))
	return err
}
func (J *EncodeWorker) getVideoParameters(inputFile string) (data *ffprobe.ProbeData, size int64, err error) {

	fileReader, err := os.Open(inputFile)
	if err != nil {
		return nil, -1, fmt.Errorf("error opening file %s because %v", inputFile, err)
	}
	stat, err := fileReader.Stat()
	if err != nil {
		return nil, 0, err
	}

	defer fileReader.Close()
	data, err = ffprobe.ProbeReader(J.ctx, fileReader)
	if err != nil {
		return nil, 0, fmt.Errorf("error getting data: %v", err)
	}
	return data, stat.Size(), nil
}
func (j *EncodeWorker) clearData(data *ffprobe.ProbeData) (container *ContainerData, err error) {
	container = &ContainerData{}

	videoStream := data.StreamType(ffprobe.StreamVideo)[0]
	container.Video = &Video{
		Id:       uint8(videoStream.Index),
		Duration: data.Format.Duration(),
	}

	betterAudioStreamPerLanguage := make(map[string]*Audio)
	for _, stream := range data.StreamType(ffprobe.StreamAudio) {
		if stream.BitRate == "" {
			stream.BitRate = "0"
		}
		bitRateInt, err := strconv.ParseUint(stream.BitRate, 10, 32) //TODO Aqui revem diferents tipos de numeros
		if err != nil {
			panic(err)
		}
		newAudio := &Audio{
			Id:             uint8(stream.Index),
			Language:       stream.Tags.Language,
			Channels:       stream.ChannelLayout,
			ChannelsNumber: uint8(stream.Channels),
			ChannelLayour:  stream.ChannelLayout,
			Default:        stream.Disposition.Default == 1,
			Bitrate:        uint(bitRateInt),
			Title:          stream.Tags.Title,
		}
		betterAudio := betterAudioStreamPerLanguage[newAudio.Language]

		//If more channels or same channels and better bitrate
		if betterAudio != nil {
			if newAudio.ChannelsNumber > betterAudio.ChannelsNumber {
				betterAudioStreamPerLanguage[newAudio.Language] = newAudio
			} else if newAudio.ChannelsNumber == betterAudio.ChannelsNumber && newAudio.Bitrate > betterAudio.Bitrate {
				betterAudioStreamPerLanguage[newAudio.Language] = newAudio
			}
		} else {
			betterAudioStreamPerLanguage[stream.Tags.Language] = newAudio
		}

	}
	for _, audioStream := range betterAudioStreamPerLanguage {
		container.Audios = append(container.Audios, audioStream)
	}

	betterSubtitleStreamPerLanguage := make(map[string]*Subtitle)
	for _, stream := range data.StreamType(ffprobe.StreamSubtitle) {
		newSubtitle := &Subtitle{
			Id:       uint8(stream.Index),
			Language: stream.Tags.Language,
			Forced:   stream.Disposition.Forced == 1,
			Comment:  stream.Disposition.Comment == 1,
			Format:   stream.CodecName,
			Title:    stream.Tags.Title,
		}

		if newSubtitle.Forced || newSubtitle.Comment {
			container.Subtitle = append(container.Subtitle, newSubtitle)
			continue
		}
		//TODO Filter Languages we don't want
		betterSubtitle := betterSubtitleStreamPerLanguage[newSubtitle.Language]
		if betterSubtitle == nil { //TODO Potser perdem subtituls que es necesiten
			betterSubtitleStreamPerLanguage[stream.Tags.Language] = newSubtitle
		} else {
			//TODO aixo es temporal per fer proves, borrar aquest else!!
			container.Subtitle = append(container.Subtitle, newSubtitle)
		}
	}
	for _, value := range betterSubtitleStreamPerLanguage {
		container.Subtitle = append(container.Subtitle, value)
	}
	return container, nil
}
func (J *EncodeWorker) FFMPEG(job *model.WorkTaskEncode, videoContainer *ContainerData, ffmpegProgressChan chan<- FFMPEGProgress) error {
	defer close(ffmpegProgressChan)
	ffmpeg := &FFMPEGGenerator{}
	ffmpeg.setInputFilters(videoContainer, job.SourceFilePath, job.WorkDir)
	ffmpeg.setVideoFilters(videoContainer)
	ffmpeg.setAudioFilters(videoContainer)
	ffmpeg.setSubtFilters(videoContainer)
	ffmpeg.setMetadata(videoContainer)
	ffmpegErrLog := ""
	ffmpegOutLog := ""
	sendObj := FFMPEGProgress{
		duration: -1,
		speed:    -1,
	}
	checkPercentageFFMPEG := func(buffer []byte, exit bool) {
		stringedBuffer := string(buffer)
		ffmpegErrLog += stringedBuffer

		duration := getDuration(stringedBuffer)
		if duration != -1 {
			sendObj.duration = duration
			sendObj.percent = float64(duration*100) / videoContainer.Video.Duration.Seconds()

		}
		speed := getSpeed(stringedBuffer)
		if speed != -1 {
			sendObj.speed = speed
		}

		if sendObj.speed != -1 && sendObj.duration != -1 {
			ffmpegProgressChan <- sendObj
			sendObj.duration = -1
			sendObj.speed = -1
		}
	}
	stdoutFFMPEG := func(buffer []byte, exit bool) {
		ffmpegOutLog += string(buffer)
	}
	sourceFileName := filepath.Base(job.SourceFilePath)
	encodedFilePath := fmt.Sprintf("%s-encoded.%s", strings.TrimSuffix(sourceFileName, filepath.Ext(sourceFileName)), "mkv")
	job.TargetFilePath = filepath.Join(job.WorkDir, encodedFilePath)

	ffmpegArguments := ffmpeg.buildArguments(uint8(J.workerConfig.Threads), job.TargetFilePath)
	//J.terminal.Log("FFMPEG Command:%s %s", helper.GetFFmpegPath(), ffmpegArguments)
	ffmpegCommand := command.NewCommandByString(helper.GetFFmpegPath(), ffmpegArguments).
		SetWorkDir(job.WorkDir).
		SetStdoutFunc(stdoutFFMPEG).
		SetStderrFunc(checkPercentageFFMPEG)

	if runtime.GOOS == "linux" {
		ffmpegCommand.AddEnv(fmt.Sprintf("LD_LIBRARY_PATH=%s", filepath.Dir(helper.GetFFmpegPath())))
	}
	exitCode, err := ffmpegCommand.RunWithContext(J.ctx)
	if err != nil {
		return fmt.Errorf("%w: stder:%s stdout:%s", err, ffmpegErrLog, ffmpegOutLog)
	}
	if exitCode != 0 {
		return fmt.Errorf("exit code %d: stder:%s stdout:%s", exitCode, ffmpegErrLog, ffmpegOutLog)
	}

	return nil
}

type ProgressTrackReader struct {
	taskTracker *TaskTracks
	io.ReadCloser
	sha hash.Hash
}

func NewProgressTrackStream(track *TaskTracks, reader io.ReadCloser) *ProgressTrackReader {
	return &ProgressTrackReader{
		taskTracker: track,
		ReadCloser:  reader,
		sha:         sha256.New(),
	}
}

func (P *ProgressTrackReader) Read(p []byte) (n int, err error) {
	n, err = P.ReadCloser.Read(p)
	P.taskTracker.Increment(n)
	P.sha.Write(p[0:n])
	return n, err
}

func (P *ProgressTrackReader) SumSha() []byte {
	return P.sha.Sum(nil)
}

func (J *EncodeWorker) UploadJob(task *model.WorkTaskEncode, track *TaskTracks) error {
	J.updateTaskStatus(task, model.UploadNotification, model.StartedNotificationStatus, "")
	err := retry.Do(func() error {
		track.UpdateValue(0)
		encodedFile, err := os.Open(task.TargetFilePath)
		if err != nil {
			return err
		}
		defer encodedFile.Close()
		fi, _ := encodedFile.Stat()
		fileSize := fi.Size()
		track.SetTotal(fileSize)
		sha := sha256.New()
		if _, err := io.Copy(sha, encodedFile); err != nil {
			return err
		}
		checksum := hex.EncodeToString(sha.Sum(nil))
		encodedFile.Seek(0, io.SeekStart)
		reader := progress.NewReader(encodedFile)

		client := &http.Client{}
		//go printProgress(J.ctx, reader, fileSize, wg, "Uploading")
		req, err := http.NewRequestWithContext(J.ctx, "POST", task.TaskEncode.UploadURL, reader)
		if err != nil {
			return err
		}
		req.ContentLength = fileSize
		req.Body = reader
		req.GetBody = func() (io.ReadCloser, error) {
			return ioutil.NopCloser(reader), nil
		}

		req.Header.Add("checksum", checksum)
		req.Header.Add("Content-Type", "application/octet-stream")
		req.Header.Add("Content-Length", strconv.FormatInt(fileSize, 10))
		resp, err := client.Do(req)

		if err != nil {
			return err
		}
		//wg.Wait()
		if resp.StatusCode != 201 {
			return fmt.Errorf("invalid status Code %d", resp.StatusCode)
		}
		track.UpdateValue(fileSize)
		return nil
	}, retry.Delay(time.Second*5),
		retry.RetryIf(func(err error) bool {
			return !errors.Is(err, context.Canceled)
		}),
		retry.DelayType(retry.FixedDelay),
		retry.Attempts(17280),
		retry.LastErrorOnly(true),
		retry.OnRetry(func(n uint, err error) {
			J.terminal.Log("Error on uploading job %s", err.Error())
		}))

	if err != nil {
		J.updateTaskStatus(task, model.UploadNotification, model.FailedNotificationStatus, "")
		return err
	}

	J.updateTaskStatus(task, model.UploadNotification, model.CompletedNotificationStatus, "")
	return nil
}

func (J *EncodeWorker) errorJob(taskEncode *model.WorkTaskEncode, err error) {
	if errors.Is(err, context.Canceled) {
		J.updateTaskStatus(taskEncode, model.JobNotification, model.CanceledNotificationStatus, "")
	} else {
		J.updateTaskStatus(taskEncode, model.JobNotification, model.FailedNotificationStatus, err.Error())
	}

	taskEncode.Clean()
}

func (J *EncodeWorker) Execute(workData []byte) error {
	taskEncode := &model.TaskEncode{}
	err := json.Unmarshal(workData, taskEncode)
	if err != nil {
		return err
	}
	workDir := filepath.Join(J.tempPath, taskEncode.Id.String())
	workTaskEncode := &model.WorkTaskEncode{
		TaskEncode: taskEncode,
		WorkDir:    workDir,
	}
	os.MkdirAll(workDir, os.ModePerm)

	J.updateTaskStatus(workTaskEncode, model.JobNotification, model.StartedNotificationStatus, "")
	J.downloadChan <- workTaskEncode
	return nil
}

func (J *EncodeWorker) Cancel() {
	J.cancelContext()
}
func (J *EncodeWorker) GetID() string {
	return J.name
}
func (J *EncodeWorker) updateTaskStatus(encode *model.WorkTaskEncode, notificationType model.NotificationType, status model.NotificationStatus, message string) {
	encode.TaskEncode.EventID++
	event := model.TaskEvent{
		Id:               encode.TaskEncode.Id,
		EventID:          encode.TaskEncode.EventID,
		EventType:        model.NotificationEvent,
		WorkerName:       J.workerConfig.Name,
		EventTime:        time.Now(),
		NotificationType: notificationType,
		Status:           status,
		Message:          message,
	}
	J.Manager.EventNotification(event)
	J.terminal.Log("[%s] %s have been %s: %s", event.Id.String(), event.NotificationType, event.Status, event.Message)

	J.saveTaskStatusDisk(&model.TaskStatus{
		LastState: &event,
		Task:      encode,
	})

}

func (J *EncodeWorker) saveTaskStatusDisk(taskEncode *model.TaskStatus) {
	J.mu.Lock()
	defer J.mu.Unlock()
	b, err := json.MarshalIndent(taskEncode, "", "\t")
	if err != nil {
		panic(err)
	}
	eventFile, err := os.OpenFile(filepath.Join(taskEncode.Task.WorkDir, fmt.Sprintf("%s.json", taskEncode.Task.TaskEncode.Id)), os.O_TRUNC|os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		return
	}
	defer eventFile.Close()
	eventFile.Write(b)
	eventFile.Sync()
}
func (J *EncodeWorker) readTaskStatusFromDiskByPath(filepath string) *model.TaskStatus {
	eventFile, err := os.Open(filepath)
	if err != nil {
		panic(err)
	}
	defer eventFile.Close()
	b, err := io.ReadAll(eventFile)
	if err != nil {
		panic(err)
	}
	taskStatus := &model.TaskStatus{}
	err = json.Unmarshal(b, taskStatus)
	if err != nil {
		panic(err)
	}
	return taskStatus
}

func (J *EncodeWorker) PGSMkvExtractDetectAndConvert(taskEncode *model.WorkTaskEncode, track *TaskTracks, container *ContainerData) error {
	var PGSTOSrt []*Subtitle
	for _, subt := range container.Subtitle {
		if subt.isImageTypeSubtitle() {
			PGSTOSrt = append(PGSTOSrt, subt)
		}
	}
	if len(PGSTOSrt) > 0 {
		J.updateTaskStatus(taskEncode, model.MKVExtractNotification, model.StartedNotificationStatus, "")
		track.Message(string(model.MKVExtractNotification))
		track.SetTotal(0)
		err := J.MKVExtract(PGSTOSrt, taskEncode)
		if err != nil {
			J.updateTaskStatus(taskEncode, model.MKVExtractNotification, model.FailedNotificationStatus, err.Error())
			return err
		}
		J.updateTaskStatus(taskEncode, model.MKVExtractNotification, model.CompletedNotificationStatus, "")

		J.updateTaskStatus(taskEncode, model.PGSNotification, model.StartedNotificationStatus, "")
		track.Message(string(model.PGSNotification))
		err = J.convertPGSToSrt(taskEncode, container, PGSTOSrt)
		if err != nil {
			J.updateTaskStatus(taskEncode, model.PGSNotification, model.FailedNotificationStatus, err.Error())
			return err
		} else {
			J.updateTaskStatus(taskEncode, model.PGSNotification, model.CompletedNotificationStatus, "")
		}
	}
	return nil
}

func (J *EncodeWorker) convertPGSToSrt(taskEncode *model.WorkTaskEncode, container *ContainerData, subtitles []*Subtitle) error {
	out := make(chan *model.TaskPGSResponse)
	var pendingPGSResponses []<-chan *model.TaskPGSResponse
	for _, subtitle := range subtitles {
		subFile, err := os.Open(filepath.Join(taskEncode.WorkDir, fmt.Sprintf("%d.sup", subtitle.Id)))
		if err != nil {
			return err
		}
		outputBytes, err := ioutil.ReadAll(subFile)
		if err != nil {
			return err
		}
		subFile.Close()
		//log.Infof("Subtitle %d is PGS, requesting  conversion...", subtitle.Id)

		PGSResponse := J.RequestPGSJob(model.TaskPGS{
			Id:          taskEncode.TaskEncode.Id,
			PGSID:       int(subtitle.Id),
			PGSdata:     outputBytes,
			PGSLanguage: subtitle.Language,
		})
		pendingPGSResponses = append(pendingPGSResponses, PGSResponse)
	}
	go func() {
		for _, c := range pendingPGSResponses {
			for v := range c {
				out <- v
			}
		}
		close(out)
	}()

	for {
		select {
		case <-J.ctx.Done():
			return J.ctx.Err()
		case <-time.After(time.Minute * 90):
			return errors.New("timeout Waiting for PGS Job Done")
		case response, ok := <-out:
			if !ok {
				return nil
			}
			if response.Err != "" {
				return fmt.Errorf("error on Process PGS %d: %s", response.PGSID, response.Err)
			}
			subtFilePath := filepath.Join(taskEncode.WorkDir, fmt.Sprintf("%d.srt", response.PGSID))
			err := ioutil.WriteFile(subtFilePath, response.Srt, os.ModePerm)
			if err != nil {
				return err
			}
		}
	}
}

func (J *EncodeWorker) MKVExtract(subtitles []*Subtitle, taskEncode *model.WorkTaskEncode) error {
	mkvExtractCommand := command.NewCommand(helper.GetMKVExtractPath(), "tracks", taskEncode.SourceFilePath).
		SetWorkDir(taskEncode.WorkDir)
	if runtime.GOOS == "linux" {
		mkvExtractCommand.AddEnv(fmt.Sprintf("LD_LIBRARY_PATH=%s", filepath.Dir(helper.GetMKVExtractPath())))
	}
	for _, subtitle := range subtitles {
		mkvExtractCommand.AddParam(fmt.Sprintf("%d:%d.sup", subtitle.Id, subtitle.Id))
	}

	_, err := mkvExtractCommand.RunWithContext(J.ctx, command.NewAllowedCodesOption(0, 1))
	if err != nil {
		J.terminal.Log("MKVExtract Command:%s", mkvExtractCommand.GetFullCommand())
		return fmt.Errorf("MKVExtract unexpected error:%v", err.Error())
		return err
	}

	return nil
}

func (J *EncodeWorker) downloadQueue() {
	for {
		select {
		case <-J.ctx.Done():
			return
		case <-time.After(time.Second):
			if len(J.encodeChan) <= 1 {
				job, ok := <-J.downloadChan
				if !ok {
					continue
				}
				taskTrack := J.terminal.AddTask(job.TaskEncode.Id.String(), DownloadJobStepType)

				J.updateTaskStatus(job, model.DownloadNotification, model.StartedNotificationStatus, "")
				time.Sleep(time.Second * 2)
				err := J.dowloadFile(job, taskTrack)
				if err != nil {
					J.updateTaskStatus(job, model.DownloadNotification, model.FailedNotificationStatus, err.Error())
					taskTrack.Error()
					J.errorJob(job, err)
					continue
				}
				J.updateTaskStatus(job, model.DownloadNotification, model.CompletedNotificationStatus, "")
				taskTrack.Done()
				J.encodeChan <- job
			}
		}
	}

}

func (J *EncodeWorker) uploadQueue() {
	for {
		select {
		case <-J.ctx.Done():
			return
		case job, ok := <-J.uploadChan:
			if !ok {
				continue
			}
			taskTrack := J.terminal.AddTask(job.TaskEncode.Id.String(), UploadJobStepType)
			err := J.UploadJob(job, taskTrack)
			if err != nil {
				taskTrack.Error()
				J.errorJob(job, err)
				continue
			}

			J.updateTaskStatus(job, model.JobNotification, model.CompletedNotificationStatus, "")
			taskTrack.Done()
			job.Clean()
		}
	}

}

func (J *EncodeWorker) encodeQueue() {
	for {
		select {
		case <-J.ctx.Done():
			return
		case job, ok := <-J.encodeChan:
			if !ok {
				continue
			}
			taskTrack := J.terminal.AddTask(job.TaskEncode.Id.String(), EncodeJobStepType)
			err := J.encodeVideo(job, taskTrack)
			if err != nil {
				taskTrack.Error()
				J.errorJob(job, err)
				continue
			}

			taskTrack.Done()
			J.uploadChan <- job
		}
	}

}

func (J *EncodeWorker) encodeVideo(job *model.WorkTaskEncode, track *TaskTracks) error {
	J.updateTaskStatus(job, model.FFProbeNotification, model.StartedNotificationStatus, "")
	track.Message(string(model.FFProbeNotification))
	sourceVideoParams, sourceVideoSize, err := J.getVideoParameters(job.SourceFilePath)
	if err != nil {
		J.updateTaskStatus(job, model.FFProbeNotification, model.FailedNotificationStatus, err.Error())
		return err
	}
	J.updateTaskStatus(job, model.FFProbeNotification, model.CompletedNotificationStatus, "")

	videoContainer, err := J.clearData(sourceVideoParams)
	if err != nil {
		J.terminal.Log("Error in clearData", J.GetID())
		return err
	}
	if err = J.PGSMkvExtractDetectAndConvert(job, track, videoContainer); err != nil {
		return err
	}
	J.updateTaskStatus(job, model.FFMPEGSNotification, model.StartedNotificationStatus, "")
	track.ResetMessage()
	track.SetTotal(10000)
	FFMPEGProgressChan := make(chan FFMPEGProgress)

	go func() {
		lastProgressEvent := int64(0)
		lastProgressUpdate := int64(0)
	loop:
		for {
			select {
			case <-J.ctx.Done():
				return
			case FFMPEGProgress, open := <-FFMPEGProgressChan:
				if !open {
					track.Increment64(10000 - lastProgressUpdate)
					break loop
				}

				percentNow := int64(FFMPEGProgress.percent * 100)
				increment := percentNow - lastProgressUpdate
				track.Increment64(increment)
				lastProgressUpdate = percentNow

				if percentNow-lastProgressEvent > 1000 {
					J.updateTaskStatus(job, model.FFMPEGSNotification, model.StartedNotificationStatus, fmt.Sprintf("{\"progress\":\"%.2f\"}", track.PercentDone()))
					lastProgressEvent = percentNow
				}
			}
		}
	}()
	err = J.FFMPEG(job, videoContainer, FFMPEGProgressChan)
	if err != nil {
		//<-time.After(time.Minute*30)
		J.updateTaskStatus(job, model.FFMPEGSNotification, model.FailedNotificationStatus, err.Error())
		return err
	}
	<-time.After(time.Second * 1)

	encodedVideoParams, encodedVideoSize, err := J.getVideoParameters(job.TargetFilePath)
	if err != nil {
		J.updateTaskStatus(job, model.FFMPEGSNotification, model.FailedNotificationStatus, err.Error())
		return err
	}
	diffDuration := encodedVideoParams.Format.DurationSeconds - sourceVideoParams.Format.DurationSeconds
	if diffDuration > 60 || diffDuration < -60 {
		err = fmt.Errorf("source File duration %f is diferent than encoded %f", sourceVideoParams.Format.DurationSeconds, encodedVideoParams.Format.DurationSeconds)
		J.updateTaskStatus(job, model.FFMPEGSNotification, model.FailedNotificationStatus, err.Error())
		return err
	}
	if encodedVideoSize > sourceVideoSize {
		err = fmt.Errorf("source File size %d bytes is less than encoded %d bytes", sourceVideoSize, encodedVideoSize)
		J.updateTaskStatus(job, model.FFMPEGSNotification, model.FailedNotificationStatus, err.Error())
		return err
	}
	J.updateTaskStatus(job, model.FFMPEGSNotification, model.CompletedNotificationStatus, "")
	return nil
}

func (J *EncodeWorker) isQueueFull() bool {
	return len(J.downloadChan) >= DOWNLOAD_QUEUE_SIZE
}

type FFMPEGGenerator struct {
	inputPaths     []string
	VideoFilter    string
	AudioFilter    []string
	SubtitleFilter []string
	Metadata       string
}

func (F *FFMPEGGenerator) setAudioFilters(container *ContainerData) {

	for index, audioStream := range container.Audios {
		//TODO que pasa quan el channelLayout esta empty??
		title := fmt.Sprintf("%s (%s)", audioStream.Language, audioStream.ChannelLayour)
		metadata := fmt.Sprintf(" -metadata:s:a:%d \"title=%s\"", index, title)
		codecQuality := fmt.Sprintf("-c:a:%d %s -vbr %d", index, "libfdk_aac", 5)
		F.AudioFilter = append(F.AudioFilter, fmt.Sprintf(" -map 0:%d %s %s", audioStream.Id, metadata, codecQuality))
	}
}
func (F *FFMPEGGenerator) setVideoFilters(container *ContainerData) {
	videoFilterParameters := fmt.Sprintf("\"scale='min(%d,iw)':min'(%d,ih)':force_original_aspect_ratio=decrease\"", 1920, 1080)
	videoEncoderQuality := fmt.Sprintf("-c:v %s -crf %d -preset %s", "libx265", 21, "medium")
	//TODO HDR??
	videoHDR := ""
	F.VideoFilter = fmt.Sprintf("-map 0:%d -map_chapters -1 -flags +global_header -filter:v %s %s %s", container.Video.Id, videoFilterParameters, videoHDR, videoEncoderQuality)

}
func (F *FFMPEGGenerator) setSubtFilters(container *ContainerData) {
	subtInputIndex := 1
	for index, subtitle := range container.Subtitle {
		if subtitle.isImageTypeSubtitle() {

			subtitleMap := fmt.Sprintf("-map %d -c:s:%d srt", subtInputIndex, index)
			subtitleForced := ""
			subtitleComment := ""
			if subtitle.Forced {
				subtitleForced = fmt.Sprintf(" -disposition:s:s:%d forced  -disposition:s:s:%d default", index, index)
			}
			if subtitle.Comment {
				subtitleComment = fmt.Sprintf(" -disposition:s:s:%d comment", index)
			}

			F.SubtitleFilter = append(F.SubtitleFilter, fmt.Sprintf("%s %s %s -metadata:s:s:%d language=%s -metadata:s:s:%d \"title=%s\" -max_interleave_delta 0", subtitleMap, subtitleForced, subtitleComment, index, subtitle.Language, index, subtitle.Title))
			subtInputIndex++
		} else {
			F.SubtitleFilter = append(F.SubtitleFilter, fmt.Sprintf("-map 0:%d -c:s:%d copy", subtitle.Id, index))
		}

	}
}
func (F *FFMPEGGenerator) setMetadata(container *ContainerData) {
	F.Metadata = fmt.Sprintf("-metadata encodeParameters='%s'", container.ToJson())
}
func (F *FFMPEGGenerator) buildArguments(threads uint8, outputFilePath string) string {
	coreParameters := fmt.Sprintf("-hide_banner  -threads %d", threads)
	inputsParameters := ""
	for _, input := range F.inputPaths {
		inputsParameters = fmt.Sprintf("%s -i \"%s\"", inputsParameters, input)
	}
	//-ss 900 -t 10
	audioParameters := ""
	for _, audio := range F.AudioFilter {
		audioParameters = fmt.Sprintf("%s %s", audioParameters, audio)
	}
	subtParameters := ""
	for _, subt := range F.SubtitleFilter {
		subtParameters = fmt.Sprintf("%s %s", subtParameters, subt)
	}

	return fmt.Sprintf("%s %s -max_muxing_queue_size 9999 %s %s %s %s %s -y", coreParameters, inputsParameters, F.VideoFilter, audioParameters, subtParameters, F.Metadata, outputFilePath)
}

func (F *FFMPEGGenerator) setInputFilters(container *ContainerData, sourceFilePath string, tempPath string) {
	F.inputPaths = append(F.inputPaths, sourceFilePath)
	inputIndex := 0
	if container.HaveImageTypeSubtitle() {
		for _, subt := range container.Subtitle {
			if subt.isImageTypeSubtitle() {
				inputIndex++
				F.inputPaths = append(F.inputPaths, filepath.Join(tempPath, fmt.Sprintf("%d.srt", subt.Id)))
			}
		}
	}
}

type Video struct {
	Id       uint8
	Duration time.Duration
}
type Audio struct {
	Id             uint8
	Language       string
	Channels       string
	ChannelsNumber uint8
	ChannelLayour  string
	Default        bool
	Bitrate        uint
	Title          string
}
type Subtitle struct {
	Id       uint8
	Language string
	Forced   bool
	Comment  bool
	Format   string
	Title    string
}
type ContainerData struct {
	Video    *Video
	Audios   []*Audio
	Subtitle []*Subtitle
}

func (C *ContainerData) HaveImageTypeSubtitle() bool {
	for _, sub := range C.Subtitle {
		if sub.isImageTypeSubtitle() {
			return true
		}
	}
	return false
}
func (C *ContainerData) ToJson() string {
	b, err := json.Marshal(C)
	if err != nil {
		panic(err)
	}
	return string(b)
}
func (C *Subtitle) isImageTypeSubtitle() bool {
	return strings.Index(strings.ToLower(C.Format), "pgs") != -1
}
