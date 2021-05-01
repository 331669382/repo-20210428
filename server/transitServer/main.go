package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	if len(os.Args) != 2 {
		fmt.Println("Args:<host:port>")
		return
	}

	r := gin.Default()
	r.POST("/register", RegisterHandler)
	r.POST("/shutdown", ShutdownHandler)
	if err := r.Run(os.Args[1]); err != nil {
		fmt.Println(err)
	}
}
