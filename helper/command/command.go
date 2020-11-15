package command

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
)

type ReaderFunc func(buffer []byte,exit bool)
type Option struct {
	PanicOnError bool
	AllowedCodes []int
}
type Command struct {
	Command string
	Params []string
	Env []string
	WorkDir string
	StdoutFunc ReaderFunc
	SterrFunc ReaderFunc

}
func NewPanicOption() Option {
	return Option{
		PanicOnError: true,
	}
}
func NewAllowedCodesOption(codes ...int) Option {
	return Option{
		AllowedCodes:codes,
	}
}
func NewCommandByString(command string,params string) *Command {
	return NewCommand(command,StringToSlice(params)...)
}
func NewCommand(command string, params ...string) *Command {
	cmd := &Command{
		Command:    command,
		Params:     params,
		Env:        os.Environ(),
		WorkDir:    GetWD(),
	}
	return cmd
}

func (C *Command) AddParam(param string) *Command{
	C.Params=append(C.Params,param)
	return C
}

func (C *Command) SetWorkDir(workDir string) *Command{
	C.WorkDir=workDir
	return C
}
func (C *Command) SetEnv(env []string) *Command{
	C.Env=env
	return C
}

func (C *Command) AddEnv(env string) *Command{
	C.Env=append(C.Env,env)
	return C
}

func (C *Command) SetStdoutFunc(StdoutFunc ReaderFunc) *Command{
	C.StdoutFunc=StdoutFunc
	return C
}

func (C *Command) SetStderrFunc(StderrtFunc ReaderFunc) *Command{
	C.SterrFunc=StderrtFunc
	return C
}
func (C *Command) Run(opt ...Option) (exitCode int, err error){
	return C.RunWithContext(context.Background(),opt...)
}

func (C *Command) RunWithContext(ctx context.Context,opt ...Option) (exitCode int, err error){
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd=exec.CommandContext(ctx,C.Command,C.Params...)
	}else{
		fullCommand := append([]string{C.Command},C.Params...)
		cmd=exec.CommandContext(ctx,"nice",append([]string{"-20"},fullCommand...)...)
	}
	cmd.Env=C.Env
	cmd.Dir=C.WorkDir
	stdout, err := cmd.StdoutPipe()
	if err!=nil {
		return
	}
	stderr, err := cmd.StderrPipe()
	if err!=nil {
		return
	}
	if err= cmd.Start();err!=nil {
		return -1,err
	}


	go C.readerStreamProcessor(ctx,stdout,C.StdoutFunc)
	go C.readerStreamProcessor(ctx,stderr,C.SterrFunc)

	/*if runtime.GOOS == "windows" {
		<-time.After(time.Second*30)
		proc:=exec.Command("wmic","process","where",fmt.Sprintf("ProcessID=\"%d\"",cmd.Process.Pid),"call","setpriority","64")
		if err:=proc.Start();err!=nil{
			return -1,err
		}
		if err:=proc.Wait();err!=nil{
			return -1,err
		}
	}*/
	err = cmd.Wait()
	if err!=nil{
		if msg, ok := err.(*exec.ExitError); ok{ // there is error code
			exitCode :=  msg.Sys().(syscall.WaitStatus).ExitStatus()
			if allowedCodes(opt,exitCode){
				return exitCode,nil
			}
			if isPanicOpt(opt){
				panic(err)
			}
			return exitCode,err
		}else{
			if isPanicOpt(opt){
				panic(err)
			}
			return -1,err
		}
	}
	return 0,nil
}

func isPanicOpt(opts []Option) bool {
	for _,opt:=range opts {
		if opt.PanicOnError {
			return true
		}
	}
	return false
}
func allowedCodes(opts []Option,exitCode int) bool {
	for _,opt:=range opts {
		if len(opt.AllowedCodes)>0 {
			for _,code:= range opt.AllowedCodes {
				if code == exitCode {
					return true
				}
			}
		}
	}
	return exitCode==0
}

func (C *Command) readerStreamProcessor(ctx context.Context,reader io.ReadCloser,callbackFunc ReaderFunc){
	buffer := make([]byte, 40)
loop:
	for {
		select{
		case <-ctx.Done():
			return
		default:
			readed, err := reader.Read(buffer)
			if err != nil {
				if err == io.EOF {
					if callbackFunc!=nil {
						callbackFunc(nil, true)
					}
				}
				break loop
			}
			if callbackFunc!=nil {
				callbackFunc(buffer[0:readed], false)
			}
		}
	}
}

func (C *Command) GetFullCommand() string {
	return fmt.Sprintf("%s %s",C.Command,strings.Join(C.Params," "))
}

func GetWD() string {
	path, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return path
}


func StringToSlice(command string) (output []string) {
	cutDoubleQuote:=true
	cutQuote:=true
	inLineWord:=""
	for _,c := range command {
		if c == ' ' && cutDoubleQuote && cutQuote {
			if len(inLineWord)>0 {
				if inLineWord[0] == '\''{
					inLineWord = strings.Trim(inLineWord,"'")
				}else if inLineWord[0] == '"'{
					inLineWord = strings.Trim(inLineWord,"\"")
				}
				output = append(output,inLineWord)
				inLineWord=""
			}
			continue
		}else if c == '"' {
			cutDoubleQuote=!cutDoubleQuote
		}else if c == '\'' {
			cutQuote=!cutQuote
		}
		inLineWord=inLineWord+string(c)
	}
	output = append(output,inLineWord)
	return output
}

