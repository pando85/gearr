package helper

import (
	"compress/gzip"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/avast/retry-go"
	"github.com/rakyll/statik/fs"
	log "github.com/sirupsen/logrus"
)

var (
	ApplicationFileName  string
	ValidVideoExtensions = []string{"mp4", "mpg", "m4a", "m4v", "f4v", "f4a", "m4b", "m4r", "f4b", "mov ", "ogg", "oga", "ogv", "ogx ", "wmv", "wma", "asf ", "webm", "avi", "flv", "vob ", "mkv"}
	STUNServers          = []string{"https://api.ipify.org?format=text", "https://ifconfig.me", "https://ident.me/", "https://myexternalip.com/raw"}
	updateURL            = "https://github.com/pando85/transcoder-server/releases/download/master/%s"
	workingDirectory     = filepath.Join(os.TempDir(), "transcoder")
	ffmpegPath           = "ffmpeg"
	mkvExtractPath       = "mkvextract"
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

func HashSha1Myself() string {
	sha1, err := GenerateSha1(os.Args[0])
	if err != nil {
		panic(err)
	}
	return sha1
}

func SetLogLevel(level string) {
	switch strings.ToLower(level) {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warning":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	default:
		log.SetLevel(log.InfoLevel)
		log.Warnf("invalid log level '%s', defaulting to 'info'", level)
	}
}
