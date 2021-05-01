package main

import (
	"io"
	"log"
	"os"
	"sync"
)

var (
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

type TempLog struct {
	logList []string
	lock    sync.Mutex
}

func (tempLog *TempLog) Write(p []byte) (n int, err error) {
	tempLog.lock.Lock()
	tempLog.logList = append(tempLog.logList, string(p))
	tempLog.lock.Unlock()
	return len(p), nil
}

func (tempLog *TempLog) ReadAndFlush() []string {
	tempLog.lock.Lock()
	s := tempLog.logList
	tempLog.logList = []string{}
	tempLog.lock.Unlock()
	return s
}

var tempLog *TempLog = &TempLog{
	lock: sync.Mutex{},
}

func SetLog() error {
	file, err := os.OpenFile("log.txt",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open error log file:", err)
		return err
	}
	Info = log.New(io.MultiWriter(file, os.Stdout, tempLog),
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Warning = log.New(io.MultiWriter(file, os.Stdout, tempLog),
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Error = log.New(io.MultiWriter(file, os.Stdout, tempLog),
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
	return nil
}
