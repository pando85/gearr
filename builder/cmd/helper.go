package cmd

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"transcoder/helper"
	"transcoder/helper/command"

	"github.com/shurcooL/vfsgen"
)

func prepareBuildEnv(path string) (buildPath string, distPath string) {
	buildPath = filepath.Join(command.GetWD(), "build", path)
	os.RemoveAll(buildPath)
	if err := os.MkdirAll(buildPath, os.ModePerm); err != nil && !os.IsExist(err) {
		panic(err)
	}
	distPath = filepath.Join(command.GetWD(), "dist")
	if err := os.MkdirAll(distPath, os.ModePerm); err != nil && !os.IsExist(err) {
		panic(err)
	}
	return buildPath, distPath
}

func getDependency() {
	command.NewCommand("go", "mod", "download").Run(command.NewPanicOption())
	command.NewCommand("go", "get", "-u", "github.com/rakyll/statik").Run(command.NewPanicOption())
}

func getCapturingGroupsRegex(r *regexp.Regexp, parse string) map[string]string {
	capturedGroups := make(map[string]string)
	names := r.SubexpNames()
	res := r.FindAllStringSubmatch(parse, -1)
	for i, _ := range res[0] {
		if i != 0 {
			capturedGroups[names[i]] = res[0][i]
		}
	}
	return capturedGroups
}

func copyResources(buildPath string, sourcePath string, GOOS string, GOARCH string) error {
	archOSRegex := regexp.MustCompile(`(?P<GOOS>(darwin|linux|windows))-(?P<GOARCH>amd64)`)
	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if archOSRegex.MatchString(path) {
			capturedGroups := getCapturingGroupsRegex(archOSRegex, path)
			if capturedGroups["GOOS"] != GOOS || capturedGroups["GOARCH"] != GOARCH {
				return nil
			}
		}
		relative, _ := filepath.Rel(sourcePath, path)
		buildPathRel := filepath.Join(buildPath, relative)
		if info.IsDir() {
			os.Mkdir(buildPathRel, os.ModePerm)
		} else {
			compressed := false
			if filepath.Ext(path) == ".gz" {
				compressed = true
				buildPathRel = strings.TrimSuffix(buildPathRel, filepath.Ext(path))
			}
			if _, err := helper.CopyFilePath(path, buildPathRel, compressed); err != nil {
				panic(err)
			}
		}

		return nil
	})
	return err
}

func Embed(resources string, target string) {
	var fs http.FileSystem = http.Dir(resources)

	err := vfsgen.Generate(fs, vfsgen.Options{Filename: filepath.Join(target, "assets_vfsdata.go")})
	if err != nil {
		panic(err)
	}

	//TODO create dir generate and copy assets_vfsdata.go
	//execute(helper.GetWD(),"go","run","github.com/rakyll/statik",fmt.Sprintf("-src=%s",resources),fmt.Sprintf("-dest=%s",target),"-f","-include","*")
}
