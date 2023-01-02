package helper

import (
	"compress/gzip"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/rakyll/statik/fs"
	log "github.com/sirupsen/logrus"
	"gopkg.in/vansante/go-ffprobe.v2"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var (
	ApplicationFileName  string
	ValidVideoExtensions = []string{"mp4", "mpg", "m4a", "m4v", "f4v", "f4a", "m4b", "m4r", "f4b", "mov ", "ogg", "oga", "ogv", "ogx ", "wmv", "wma", "asf ", "webm", "avi", "flv", "vob ", "mkv"}
	STUNServers          = []string{"https://api.ipify.org?format=text", "https://ifconfig.me", "https://ident.me/", "https://myexternalip.com/raw"}
	updateURL            = "https://github.com/segator/transcoderd/releases/download/wip-master/%s"
	workingDirectory     = filepath.Join(os.TempDir(), "transcoder")
	ffmpegPath           = ""
	mkvExtractPath       = ""
)

func ValidExtension(extension string) bool {
	for _, validExtension := range ValidVideoExtensions {
		if extension == validExtension {
			return true
		}
	}
	return false
}

func CheckPath(path string) {
	if !filepath.IsAbs(path) {
		log.Panicf("download-path %s must be absolute and ends with /", path)
	}
}

func GetPublicIP() (publicIP string) {
	retry.Do(func() error {
		randomIndex := rand.Intn(len(STUNServers))
		resp, err := http.Get(STUNServers[randomIndex])
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		publicIPBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		publicIP = string(publicIPBytes)
		return nil
	}, retry.Delay(time.Millisecond*100), retry.Attempts(360), retry.LastErrorOnly(true))
	return publicIP
}

func NameCleaner(path string) string {
	//TODO millor una regex
	path = strings.Replace(path, "h264", "h265", -1)
	path = strings.Replace(path, "H264", "H265", -1)
	path = strings.Replace(path, "x264", "x265", -1)
	path = strings.Replace(path, "X264", "X265", -1)
	return path
}

func GetWorkingDir() string {
	return workingDirectory
}

func CopyFilePath(src, dst string, compressed bool) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()
	var reader io.Reader
	reader = source
	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()

	if compressed {
		reader, err = gzip.NewReader(source)
		if err != nil {
			return 0, nil
		}
	}
	nBytes, err := io.Copy(destination, reader)
	return nBytes, err
}
func DisembedFile(embedFS http.FileSystem, statikPath string, targetFilePath string) (string, error) {
	embededFile, err := embedFS.Open(statikPath)
	if err != nil {
		panic(err)
	}
	defer embededFile.Close()
	if st, _ := embededFile.Stat(); st.IsDir() {
		err := fs.Walk(embedFS, statikPath, func(path string, info os.FileInfo, err error) error {
			//I Dont have time for recurisve
			if info.IsDir() {
				return nil
			}
			_, err = DisembedFile(embedFS, fmt.Sprintf("%s/%s", statikPath, info.Name()), info.Name())
			return err
		})
		return GetWorkingDir(), err
	}

	tempPath := GetWorkingDir()
	err = os.MkdirAll(tempPath, os.ModePerm)
	if err != nil {
		return "", err
	}
	targetCopyFile := filepath.Join(tempPath, targetFilePath)
	ffProbeFile, err := os.OpenFile(targetCopyFile, os.O_TRUNC|os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		return "", err
	}
	defer ffProbeFile.Close()
	if _, err := io.Copy(ffProbeFile, embededFile); err != nil {
		return "", err
	}
	return targetCopyFile, nil
}

func DesembedFSFFProbe(embedFS http.FileSystem) error {
	ffprobeFile := "ffprobe"
	if runtime.GOOS == "windows" {
		ffprobeFile = ffprobeFile + ".exe"
	}
	ffprobePath, err := DisembedFile(embedFS, fmt.Sprintf("/ffprobe/%s-%s/%s", runtime.GOOS, runtime.GOARCH, ffprobeFile), ffprobeFile)
	if err != nil {
		return err
	}

	ffprobe.SetFFProbeBinPath(ffprobePath)
	return nil
}

func DesembedFFmpeg(embedFS http.FileSystem) error {
	ffmpegFileName := "ffmpeg"
	if runtime.GOOS == "windows" {
		ffmpegFileName = ffmpegFileName + ".exe"
	}
	DisembedPath, err := DisembedFile(embedFS, fmt.Sprintf("/ffmpeg/%s-%s", runtime.GOOS, runtime.GOARCH), ffmpegFileName)
	if err != nil {
		return err
	}
	setFFmpegPath(filepath.Join(DisembedPath, ffmpegFileName))
	return nil
}

func DesembedMKVExtract(embedFS http.FileSystem) error {
	mkvExtractFileName := "mkvextract"
	if runtime.GOOS == "windows" {
		mkvExtractFileName = mkvExtractFileName + ".exe"
	}
	DisembedPath, err := DisembedFile(embedFS, fmt.Sprintf("/mkvextract/%s-%s", runtime.GOOS, runtime.GOARCH), mkvExtractFileName)
	if err != nil {
		return err
	}
	setMKVExtractPath(filepath.Join(DisembedPath, mkvExtractFileName))
	return nil
}

func setFFmpegPath(newFFmpegPath string) {
	ffmpegPath = newFFmpegPath
}
func setMKVExtractPath(newMKVExtractPath string) {
	mkvExtractPath = newMKVExtractPath
}

func GetFFmpegPath() string {
	return ffmpegPath
}

func GetMKVExtractPath() string {
	return mkvExtractPath
}

func GenerateSha1(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	hashInBytes := hash.Sum(nil)[:20]
	sha1String := hex.EncodeToString(hashInBytes)
	return sha1String, nil
}

func GenerateSha1File(path string) error {
	sha1, err := GenerateSha1(path)
	if err != nil {
		return err
	}
	sha1PathFile := fmt.Sprintf("%s.sha1", path)
	w, err := os.Create(sha1PathFile)
	if err != nil {
		return err
	}
	defer w.Close()

	w.WriteString(sha1)
	return nil
}

func GetGitHubLatestVersion() string {
	var data []byte
	err := retry.Do(func() error {
		downloadSHA1URL := fmt.Sprintf(updateURL+".sha1", ApplicationFileName)
		resp, err := http.Get(downloadSHA1URL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		data, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return nil
	}, retry.Delay(time.Second*5), retry.Attempts(300), retry.LastErrorOnly(true))
	if err != nil {
		panic(err)
	}
	return string(data)
}

func HashSha1Myself() string {
	sha1, err := GenerateSha1(os.Args[0])
	if err != nil {
		panic(err)
	}
	return sha1
}

func DownloadAppLatestVersion() string {
	// Create the file
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	RunFile, err := ioutil.TempFile("", fmt.Sprintf("transcoderw*%s", ext))

	if err != nil {
		panic(err)
	}
	defer RunFile.Close()

	// Get the data
	resp, err := http.Get(fmt.Sprintf(updateURL, ApplicationFileName))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		panic(fmt.Errorf("bad status: %s", resp.Status))
	}

	// Writer the body to file
	_, err = io.Copy(RunFile, resp.Body)
	if err != nil {
		panic(err)
	}
	RunFile.Chmod(os.ModePerm)
	return RunFile.Name()
}

func IsApplicationUpToDate() bool {
	//return true
	return HashSha1Myself() == GetGitHubLatestVersion()
}
