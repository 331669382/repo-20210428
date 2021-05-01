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

func main() {
	var _config common.Config
	file, err := os.Open("config.config")
	if err != nil {
		fmt.Printf("Open ./config.config failed [Err:%v]", err)
		return
	}

	err = json.NewDecoder(file).Decode(&_config)
	if err != nil {
		fmt.Println("invalid config file")
		return
	}
	if err := log.SetLog(); err != nil {
		fmt.Printf("SetLog err:%v\n", err)
		return
	}
	httpAddr := _config.Clientlisten
	wsAddr := _config.Robotlisten

	httpErrChan := make(chan error)
	http.RunHttpServer(_config, httpAddr, httpErrChan)
	wsErrChan := make(chan error)
	err = websocket.RunWsServer(_config, wsAddr, wsErrChan)
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
