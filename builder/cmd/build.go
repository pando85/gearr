package cmd

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
	"transcoder/helper"
	"transcoder/helper/command"
)
var allPlatforms = []string{"windows-amd64","linux-amd64","darwin-amd64"}
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
		slice, _ := cmd.Flags().GetStringSlice("platform")
		if slice[0]=="all" {
			buildServer(allPlatforms)
			return
		}else{
			buildServer(cleanPlatforms(slice))
		}
	},
}

var buildWorkerCmd = &cobra.Command{
	Use:   "worker",
	Short: "w",
	Long:  `worker build`,
	Run: func(cmd *cobra.Command, args []string) {
		mode, _ := cmd.Flags().GetString("mode")
		slice, _ := cmd.Flags().GetStringSlice("platform")
		if slice[0]=="all" {
			buildWorker(allPlatforms,mode)
			return
		}else{
			buildWorker(cleanPlatforms(slice),mode)
		}
	},
}

func cleanPlatforms(platforms []string) []string{
	for i,platform := range platforms {
		if strings.Contains(platform,"macos") {
			platforms[i]=strings.Replace(platform,"macos","darwin",-1)
		}
		if strings.Contains(platform,"ubuntu") {
			platforms[i]=strings.Replace(platform,"ubuntu","linux",-1)
		}
	}
	return platforms
}

func buildServer(platforms []string) {
	log.Infof("Get Dependencies...")
	getDependency()

	for _,platform := range platforms {
		log.Infof("====== %s ======",strings.ToUpper(platform))
		pltSplit := strings.Split(platform,"-")
		GOOS:= pltSplit[0]
		GOARCH:= pltSplit[1]
		log.Infof("[%s] Preparing Build Environment...",platform)
		buildPath,distPath := prepareBuildEnv("server")
		log.Infof("[%s] Copy Resources...",platform)
		copyServerResources(buildPath,GOOS,GOARCH)
		log.Infof("[%s] Embedding resources...",platform)

		Embed(buildPath,filepath.Join(command.GetWD(),"server"))

		envs := os.Environ()
		envs = append(envs, fmt.Sprintf("GOARCH=%s",GOARCH))
		envs = append(envs, fmt.Sprintf("GOOS=%s",GOOS))
		envs = append(envs, "CGO_ENABLED=0")
		extension :=""
		if GOOS == "windows" {
			extension=".exe"
		}
		log.Infof("[%s] Building executable...",platform)
		fileName:=fmt.Sprintf("transcoderd-%s%s",platform,extension)
		outputBinPath :=fmt.Sprintf("%s/%s",distPath,fileName)
		command.NewCommand("go","build","-ldflags", fmt.Sprintf("-X main.ApplicationFileName=%s",fileName),"-o",outputBinPath).
			SetWorkDir(filepath.Join(command.GetWD(),"server")).
			SetEnv(envs).Run(command.NewPanicOption())
		helper.GenerateSha1File(outputBinPath)


	}
}

func copyServerResources(buildPath string,GOOS,GOARCH string) {
	serverResourcesPath := filepath.Join(command.GetWD(),"server","resources")
	copyResources(buildPath, serverResourcesPath,GOOS,GOARCH)
}



func buildWorker(platforms []string, buildMode string) {
	log.Infof("Get Dependencies...")
	getDependency()

	for _,platform := range platforms {
		log.Infof("====== %s ======",strings.ToUpper(platform))
		pltSplit := strings.Split(platform,"-")
		GOOS:= pltSplit[0]
		GOARCH:= pltSplit[1]

		log.Infof("[%s] Preparing Build Environment...",platform)
		buildPath,distPath := prepareBuildEnv("worker")

		log.Infof("[%s] Copy Resources...",platform)
		copyWorkerResources(buildPath,buildMode,GOOS,GOARCH)
		log.Infof("[%s] Embedding resources...",platform)
		Embed(buildPath,filepath.Join(command.GetWD(),"worker"))
		envs := os.Environ()
		envs = append(envs, fmt.Sprintf("GOARCH=%s",GOARCH))
		envs = append(envs, fmt.Sprintf("GOOS=%s",GOOS))
		extension :=""
		if GOOS == "windows" {
			envs = append(envs, "CGO_ENABLED=0")
			extension=".exe"
		} else if GOOS == "linux" {
			envs = append(envs, "CGO_ENABLED=1")
		} else if GOOS == "darwin" {
			envs = append(envs, "CGO_ENABLED=1")
			envs = append(envs,"GO111MODULE=on")
		}
		log.Infof("[%s] Building executable...",platform)
		fileName := fmt.Sprintf("transcoderw-%s-%s%s",buildMode,platform,extension)
		outputBinPath := fmt.Sprintf("%s/%s",distPath,fileName)
		print := func(buffer []byte, exit bool){
			fmt.Print(string(buffer))
		}
		command.NewCommand("go","build","-ldflags",fmt.Sprintf("-X main.ApplicationFileName=%s", fileName),"-o",outputBinPath).
			SetWorkDir(filepath.Join(command.GetWD(),"worker")).
			SetStdoutFunc(print).
			SetStderrFunc(print).
			SetEnv(envs).Run(command.NewPanicOption())

		helper.GenerateSha1File(outputBinPath)
	}
}



func copyWorkerResources(buildPath string,buildMode string,GOOS,GOARCH string) {
	workerResourcesPath := filepath.Join(command.GetWD(),"worker","resources")
	copyResources(buildPath, workerResourcesPath,GOOS,GOARCH)
	if buildMode == "gui" {
		sysF, err := os.OpenFile(filepath.Join(buildPath,"systray.enabled"),os.O_TRUNC|os.O_CREATE|os.O_RDWR, os.ModePerm)
		if err!=nil {
			panic(err)
		}
		defer sysF.Close()
		sysF.WriteString("1")
		sysF.Sync()
	}
}



func init() {
	buildCmd.AddCommand(buildServerCmd,buildWorkerCmd)
	RootCmd.AddCommand(buildCmd)

	buildCmd.PersistentFlags().StringSliceP("platform","p", []string{"all"}, "select all platforms that you want to build (all,windows-amd64,linux-amd64,darwin-amd64,linux-arm,...)")
	buildCmd.PersistentFlags().StringP("mode","m", "gui", "Build for gui or console")
}