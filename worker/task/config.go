package task

import (
	"fmt"
	"strconv"
	"strings"
	"transcoder/model"

	log "github.com/sirupsen/logrus"
	"gopkg.in/errgo.v2/errors"
)

type AcceptedJobs []model.JobType

func (A AcceptedJobs) IsAccepted(jobType model.JobType) bool {
	log.Debugf("accepted jobs: %+v", A)

	for _, j := range A {
		log.Debugf("check %s == %s", j, jobType)
		if j == jobType {
			return true
		}
	}
	return false
}

type TimeHourMinute struct {
	Hour   int
	Minute int
}

func (t *TimeHourMinute) Type() string {
	return "TimeHourMinute"
}
func (t *TimeHourMinute) String() string {
	return fmt.Sprintf("%02d:%02d", t.Hour, t.Minute)
}

func (t *TimeHourMinute) Set(value string) error {
	HourMinuteSlice := strings.Split(value, ":")
	if len(HourMinuteSlice) != 2 {
		return errors.New(fmt.Sprintf("%s is not a TimeHour", value))
	}
	n, err := strconv.Atoi(HourMinuteSlice[0])
	if err != nil {
		return err
	}
	t.Hour = n
	n, err = strconv.Atoi(HourMinuteSlice[1])
	if err != nil {
		return err
	}
	t.Minute = n
	return nil
}

type Config struct {
	UpdateMode        bool           `mapstructure:"updateMode"`
	TemporalPath      string         `mapstructure:"temporalPath"`
	Name              string         `mapstructure:"name"`
	Threads           int            `mapstructure:"threads"`
	Jobs              AcceptedJobs   `mapstructure:"acceptedJobs"`
	EncodeJobs        int            `mapstructure:"encodeJobs"`
	PgsJobs           int            `mapstructure:"pgsJobs"`
	Priority          int            `mapstructure:"priority"`
	StartAfter        TimeHourMinute `mapstructure:"startAfter"`
	StopAfter         TimeHourMinute `mapstructure:"stopAfter"`
	Paused            bool
	PGSTOSrtDLLPath   string `mapstructure:"pgsToSrtDLLPath"`
	TesseractDataPath string `mapstructure:"tesseractDataPath"`
	DotnetPath        string `mapstructure:"dotnetPath"`
}

func (c Config) HaveSettedPeriodTime() bool {
	return c.StartAfter.Hour != 0 || c.StopAfter.Hour != 0
}
