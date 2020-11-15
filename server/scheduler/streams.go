package scheduler

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"os"
	"transcoder/model"
)

type PathChecksum struct{
	path string
	checksum string
}
type JobStream struct {
	hasher hash.Hash
	video *model.Video
	path string
	file *os.File
	checksumPublisher chan PathChecksum
}

type UploadJobStream struct {
	*JobStream
}

type DownloadJobStream struct {
	*JobStream
	FileSize int64
	FileName string
}

func(U *JobStream) hash(p []byte) (err error) {
	if U.hasher==nil{
		U.hasher=sha256.New()
	}
	if _,err:=U.hasher.Write(p); err!=nil{
		return err
	}
	return nil
}
func(U *JobStream) GetHash() string{
	return hex.EncodeToString(U.hasher.Sum(nil))
}
func(U *UploadJobStream) Write(p []byte) (n int, err error) {
	U.hash(p)
	return U.file.Write(p)
}
func(D *DownloadJobStream) Read(p []byte) (n int, err error){
	readed, err := D.file.Read(p)
	if err!=nil{
		return readed,err
	}
	D.hash(p[0:readed])
	return readed,err
}

func (D *DownloadJobStream) Size() int64 {
	return D.FileSize
}

func (D *DownloadJobStream) Name() string {
	return D.FileName
}

func(U *JobStream) Close(pushChecksum bool) error{
	U.file.Sync()
	U.file.Close()
	if U.hasher!=nil && pushChecksum {
		U.checksumPublisher<-PathChecksum{
			path: U.path,
			checksum: U.GetHash(),
		}
	}
	return nil
}
func(U *UploadJobStream) Clean() error{
	return os.Remove(U.path)
}
