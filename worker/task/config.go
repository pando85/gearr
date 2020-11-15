package task

import (
	"fmt"
	"gopkg.in/errgo.v2/errors"
	"strconv"
	"strings"
	"transcoder/model"
)

type AcceptedJobs []model.JobType

func (A AcceptedJobs) IsAccepted(jobType model.JobType) bool{
	for _,j := range A {
		if j == jobType{
			return true
		}
	}
	return false
}
type TimeHourMinute struct {
	Hour int
	Minute int
}
func (t *TimeHourMinute) Type() string{
	return "TimeHourMinute"
}
func (t *TimeHourMinute) String() string {
	return fmt.Sprintf("%02d:%02d",t.Hour,t.Minute)
}

func (t *TimeHourMinute) Set(value string) error {
	HourMinuteSlice := strings.Split(value, ":")
	if len(HourMinuteSlice) != 2 {
		return errors.New(fmt.Sprintf("%s is not a TimeHour",value))
	}
	n,err:=strconv.Atoi(HourMinuteSlice[0])
	if err!=nil{
		return err
	}
	t.Hour=n
	n,err=strconv.Atoi(HourMinuteSlice[1])
	if err!=nil{
		return err
	}
	t.Minute=n
	return nil
}

type Config struct {
	NoUpdateMode        bool `mapstructure:"noUpdateMode", envconfig:"WORKER_NOUPDATE"`
	TemporalPath      string `mapstructure:"temporalPath", envconfig:"WORKER_TMP_PATH"`
	Name              string `mapstructure:"name", envconfig:"WORKER_NAME"`
	Threads           int `mapstructure:"threads", envconfig:"WORKER_THREADS"`
	Jobs              AcceptedJobs   `mapstructure:"acceptedJobs", envconfig:"WORKER_ACCEPTED_JOBS"`
	EncodeJobs        int            `mapstructure:"encodeJobs", envconfig:"WORKER_ENCODE_JOBS"`
	PgsJobs           int            `mapstructure:"pgsJobs", envconfig:"WORKER_PGS_JOBS"`
	Priority          int            `mapstructure:"priority", envconfig:"WORKER_PRIORITY"`
	StartAfter        TimeHourMinute `mapstructure:"startAfter", envconfig:"WORKER_START_AFTER"`
	StopAfter         TimeHourMinute `mapstructure:"stopAfter", envconfig:"WORKER_STOP_AFTER"`
	Paused            bool
	PGSTOSrtDLLPath   string `mapstructure:"pgsToSrtDLLPath", envconfig:"WORKER_PGS_TO_SRT_DLL_PATH"`
	TesseractDataPath string `mapstructure:"tesseractDataPath", envconfig:"WORKER_TESSERACT_DATA_PATH"`
	DotnetPath        string `mapstructure:"dotnetPath", envconfig:"WORKER_DOTNET_PATH"`
}

func (c Config) HaveSettedPeriodTime() bool {
	return c.StartAfter.Hour!=0 || c.StopAfter.Hour!=0
}
