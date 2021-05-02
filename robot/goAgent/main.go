package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

var _config Config

func main() {
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
	if err := SetLog(); err != nil {
		fmt.Printf("SetLog error:%v\n", err)
		return
	}
	r := gin.Default()
	conn := r.Group("/conn")
	{
		conn.POST("/config", ConfigHandler)
		conn.POST("/connect", ConnectHandler)
		conn.POST("/stat", StatHandler)
	}
	r.GET("/statu", statuHandler)
	r.POST("/event", EvenHandler)
	r.POST("/shutdown", ShutdownHandler)
	r.POST("/test", TestHandler)
	if err := r.Run(fmt.Sprintf("0.0.0.0:%d", _config.AgentListenPort)); err != nil {
		fmt.Println(err)
	}
}
