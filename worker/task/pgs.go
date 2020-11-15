package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"transcoder/helper/command"
	"transcoder/model"
)
var langMapping []PGSTesseractLanguage
type PGSWorker struct {
	workerConfig  Config
	tempPath      string
	cancelContext context.CancelFunc
	ctx           context.Context
	name          string
	Manager       model.Manager
	task          model.TaskPGS
}

type PGSTesseractLanguage struct{
	tessLanguage string
	mappingLanguage []string
}

func init(){
	langMapping=append(langMapping,PGSTesseractLanguage{"deu",[]string{"ger","ge","de"}})
	langMapping=append(langMapping,PGSTesseractLanguage{"eng",[]string{"en","uk"}})
	langMapping=append(langMapping,PGSTesseractLanguage{"spa",[]string{"es","esp"}})
	langMapping=append(langMapping,PGSTesseractLanguage{"fra",[]string{"fre"}})
	langMapping=append(langMapping,PGSTesseractLanguage{"chi_tra",[]string{"chi"}})
}
func NewPGSWorker(ctx context.Context, workerConfig Config, workerName string)  *PGSWorker {
	newCtx, cancel := context.WithCancel(ctx)
	tempPath := filepath.Join(workerConfig.TemporalPath, fmt.Sprintf("worker-%s", workerName))
	encodeWorker := &PGSWorker{
		name:          workerName,
		ctx:           newCtx,
		cancelContext: cancel,
		workerConfig:  workerConfig,
		tempPath:      tempPath,
	}
	return encodeWorker
}

func (P PGSWorker) IsTypeAccepted(jobType string) bool {
	return jobType == string(model.PGSToSrtJobType)
}

func (P *PGSWorker) Prepare(workData []byte, queueManager model.Manager) error {
	pgsTask := &model.TaskPGS{}
	err := json.Unmarshal(workData, pgsTask)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(P.tempPath, os.ModePerm); err != nil {
		return err
	}
	P.Manager = queueManager
	P.task = *pgsTask
	return nil
}

func (P *PGSWorker) Execute() (err error) {
	log.Infof("Converting PGS To Srt for Job %s stream %d",P.task.Id.String(),P.task.PGSID)
	//TODO events??
	inputFilePath:= filepath.Join(P.tempPath,strconv.Itoa(P.task.PGSID)+".sup")
	outputFileName:= strconv.Itoa(P.task.PGSID)+".srt"
	outputFilePath:= filepath.Join(P.tempPath,outputFileName)
	var outputBytes []byte
	defer func(){
		errString:=""
		if err!=nil{
			errString=err.Error()
		}

		pgsTaskResponse := model.TaskPGSResponse{
			Id:    P.task.Id,
			PGSID: P.task.PGSID,
			Srt:   outputBytes,
			Err:  errString,
			Queue: P.task.ReplyTo,
		}
		P.Manager.ResponsePGSJob(pgsTaskResponse)
	}()

	err = ioutil.WriteFile(inputFilePath,P.task.PGSdata,os.ModePerm)
	if err!=nil {
		return err
	}

	language := calculateTesseractLanguage(P.task.PGSLanguage)
	//<-time.After(time.Minute*30)
	PGSToSrtCommand:=command.NewCommand(P.workerConfig.DotnetPath,fmt.Sprintf("%s",P.workerConfig.PGSTOSrtDLLPath),"--input",inputFilePath,"--output",outputFilePath,"--tesseractlanguage",language,"--tesseractdata",P.workerConfig.TesseractDataPath).
		SetWorkDir(P.tempPath)
	log.Debugf("PGSTOSrt Command: %s",PGSToSrtCommand.GetFullCommand())
	ecode,err := PGSToSrtCommand.RunWithContext(P.ctx)
	if err!=nil {
		return err
	}
	if ecode!=0 {
		return errors.New(fmt.Sprintf("PGSToSrt invalid exit code %d",ecode))
	}
	f, err := os.Open(outputFilePath)
	if err!=nil {
		return err
	}
	defer f.Close()
	outputBytes,err = ioutil.ReadAll(f)
	log.Infof("Converted PGS To Srt for Job %s stream %d",P.task.Id.String(),P.task.PGSID)
	return err
}

func calculateTesseractLanguage(language string) string {
	for _,mapping := range langMapping{
		for _,mapLang := range mapping.mappingLanguage{
			if language == mapLang{
				return mapping.tessLanguage
			}
		}
	}
	return language
}

func (P *PGSWorker) Clean() error {
	log.Warnf("[%s] Cleaning up worker workspace...", P.GetID())
	err := os.RemoveAll(P.tempPath)
	if err != nil {
		log.Error("error in clean folder", P.GetID())
		return err
	}
	return nil
}

func (P *PGSWorker) Cancel() {
	log.Warnf("[%s] Canceling job %s...", P.GetID(), P.task.Id.String())
	P.cancelContext()
	//TODO hauriem de esperar de alguna manera a asegurar que el Execute() ha sortit..
}

func (P PGSWorker) GetID() string {
	return P.name
}

func (P PGSWorker) GetTaskID() uuid.UUID {
	return P.task.Id
}

func (P PGSWorker) AcceptJobs() bool {
	return true
}