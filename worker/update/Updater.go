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
		log.Info("checking for updates")
		for {
			sha1Hash := helper.HashSha1Myself()
			sha1LatestHash := helper.GetGitHubLatestVersion()
			U.binaryPath = os.Args[0]
			if sha1Hash != sha1LatestHash {
				log.Warn("application is outdated..")
				log.Info("downloading latest version..")
				U.binaryPath = helper.DownloadAppLatestVersion()
			} else {
				log.Info("already up to date")
			}
			arguments := os.Args[1:]
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
			log.Info("cleaning before close")
			err := os.Remove(U.binaryPath)
			if err == nil {
				break
			}
			<-time.After(time.Millisecond * 100)
		}
	}
}
