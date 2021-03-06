package main

import (
	"encoding/json"
	"fmt"
	"os"

	"mcurobot.com/registerServer/common"
	"mcurobot.com/registerServer/http"
	"mcurobot.com/registerServer/log"
	"mcurobot.com/registerServer/websocket"
)
const (
	ConfigFile = "registerServer.config"
)

func main() {
	file, err := os.Open(ConfigFile)
	if err != nil {
		fmt.Printf("Open %s failed [Err:%v]", ConfigFile, err)
		return
	}

	err = json.NewDecoder(file).Decode(&common.RegistryConfig)
	if err != nil {
		fmt.Println("invalid config file")
		return
	}
	if err := log.SetLog(); err != nil {
		fmt.Printf("SetLog err:%v\n", err)
		return
	}

	httpErrChan := make(chan error)
	http.RunHttpServer(httpErrChan)
	wsErrChan := make(chan error)
	err = websocket.RunWsServer(wsErrChan)
	if err != nil {
		fmt.Printf("RunWsServer err:%v\n", err)
		return
	}
	select {
	case err := <-httpErrChan:
		fmt.Printf("RunHttpServer err:%v\n", err)
		return
	case err := <-wsErrChan:
		fmt.Printf("RunWsServer err:%v\n", err)
		return
	}
}
