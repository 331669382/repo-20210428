package main

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {

	if len(os.Args) != 2 {
		fmt.Println("Args:<host:port>")
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
	r.POST("/event", EvenHandler)
	r.POST("/shutdown", ShutdownHandler)
	r.POST("/test", TestHandler)
	if err := r.Run(os.Args[1]); err != nil {
		fmt.Println(err)
	}
}
