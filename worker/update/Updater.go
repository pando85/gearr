package update

import (
	"context"
	"os"
	"sync"
	"time"
	"transcoder/helper"
	"transcoder/helper/command"

	log "github.com/sirupsen/logrus"
)

type Updater struct {
	binaryPath string
}

func NewUpdater() *Updater {
	updater := &Updater{}
	return updater
}

func (U *Updater) Run(wg *sync.WaitGroup, ctx context.Context) {
	wg.Add(1)
	go func() {
		log.Info("Checking for Updates...")
		for {
			sha1Hash := helper.HashSha1Myself()
			sha1LatestHash := helper.GetGitHubLatestVersion()
			U.binaryPath = os.Args[0]
			if sha1Hash != sha1LatestHash {
				log.Warn("Application is outdated..")
				log.Info("Downloading latest version..")
				U.binaryPath = helper.DownloadAppLatestVersion()
			} else {
				log.Info("Already up to date")
			}
			arguments := os.Args[1:]
			arguments = append(arguments, "--worker.noUpdateMode")
			ecode, err := command.NewCommand(U.binaryPath, arguments...).
				SetStderrFunc(func(buffer []byte, exit bool) {
					os.Stderr.Write(buffer)
				}).
				SetStdoutFunc(func(buffer []byte, exit bool) {
					os.Stdout.Write(buffer)
				}).RunWithContext(ctx, command.NewAllowedCodesOption(1))
			if err != nil {
				panic(err)
			}
			if ecode != 1 {
				break
			}
		}
	}()
	go func() {
		<-ctx.Done()
		U.stop()
		wg.Done()
	}()
}

func (U *Updater) stop() {
	if U.binaryPath != "" {
		for {
			log.Info("Cleaning before close...")
			err := os.Remove(U.binaryPath)
			if err == nil {
				break
			}
			<-time.After(time.Millisecond * 100)
		}
	}
}
