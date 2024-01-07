package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"transcoder/helper"
	"transcoder/helper/command"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var allPlatforms = []string{"windows-amd64", "linux-amd64", "darwin-amd64"}
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "b",
	Long:  `Build server or worker`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("build called")
	},
}

var buildServerCmd = &cobra.Command{
	Use:   "server",
	Short: "s",
	Long:  `server build`,

	Run: func(cmd *cobra.Command, args []string) {
		buildServer()
	},
}

var buildWorkerCmd = &cobra.Command{
	Use:   "worker",
	Short: "w",
	Long:  `worker build`,
	Run: func(cmd *cobra.Command, args []string) {
		buildWorker()
	},
}

func buildTarget(targetName string, isWorker bool) {
	log.Infof("Get Dependencies...")
	getDependency()

	goos := os.Getenv("GOOS")
	if goos == "" {
		goos = runtime.GOOS
	}

	goarch := os.Getenv("GOARCH")
	if goarch == "" {
		goarch = runtime.GOARCH
	}

	platform := fmt.Sprintf("%s-%s", goos, goarch)
	log.Infof("====== %s ======", platform)
	log.Infof("[%s] Preparing Build Environment...", platform)
	buildPath, distPath := prepareBuildEnv(targetName)

	log.Infof("[%s] Copy Resources...", platform)
	copyResources(buildPath, getResourcePath(targetName), goos, goarch)
	log.Infof("[%s] Embedding resources...", platform)

	Embed(buildPath, filepath.Join(command.GetWD(), targetName))

	envs := os.Environ()

	if isWorker {
		if goos == "windows" {
			envs = append(envs, "CGO_ENABLED=0")
		} else {
			envs = append(envs, "CGO_ENABLED=1")
			if goos == "darwin" {
				envs = append(envs, "GO111MODULE=on")
			}
		}
	} else {
		envs = append(envs, "CGO_ENABLED=0")
	}

	log.Infof("[%s] Building executable...", platform)

	binSuffix := "d"
	if targetName == "worker" {
		binSuffix = "w"
	}

	extension := ""
	if goos == "windows" {
		extension = ".exe"
	}

	fileName := fmt.Sprintf("transcoder%s-%s%s", binSuffix, platform, extension)
	outputBinPath := fmt.Sprintf("%s/%s", distPath, fileName)

	print := func(buffer []byte, exit bool) {
		if output := string(buffer); output != "" {
			log.Info(output)
		}
	}
	printErr := func(buffer []byte, exit bool) {
		if output := string(buffer); output != "" {
			log.Error(output)
		}
	}

	cmd := command.NewCommand("go", "build", "-ldflags", fmt.Sprintf("-X main.ApplicationFileName=%s", fileName), "-o", outputBinPath).
		SetWorkDir(filepath.Join(command.GetWD(), targetName)).
		SetEnv(envs).
		SetStdoutFunc(print).
		SetStderrFunc(printErr)
	cmd.Run(command.NewPanicOption())

	helper.GenerateSha1File(outputBinPath)
}

func buildServer() {
	buildTarget("server", false)
}

func buildWorker() {
	buildTarget("worker", true)
}

func getResourcePath(targetName string) string {
	return filepath.Join(command.GetWD(), targetName, "resources")
}

func init() {
	buildCmd.AddCommand(buildServerCmd, buildWorkerCmd)
	RootCmd.AddCommand(buildCmd)
}
