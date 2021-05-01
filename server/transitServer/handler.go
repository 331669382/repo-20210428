package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/gin-gonic/gin"
)

var baseport int = 46000
var key string = "123"
var portUse map[int]context.CancelFunc = make(map[int]context.CancelFunc)
var portMutex sync.Mutex
var CTX, CANCEL = context.WithCancel(context.Background())

func RegisterHandler(c *gin.Context) {
	j := RegisterReq{}
	err := c.ShouldBindJSON(&j)
	fmt.Println(j)
	if err != nil {
		c.JSON(200, NormalRsp{
			Code: 400,
			Msg:  err.Error(),
		})
		return
	}
	if j.Token != key {
		c.JSON(200, NormalRsp{
			Code: 400,
			Msg:  "wrong token",
		})
		return
	}
	var ctx, cancel = context.WithCancel(CTX)
	dispatcher := newDispatcher(j.ChannelInfo.Channels)
	go dispatchWorker(ctx, dispatcher, j.ChannelInfo.Channels)
	portResponse := PortResponse{
		Code: 200,
	}

	for _, channel := range j.ChannelInfo.Channels {
		port := 0
		if port, err = startListen(ctx, cancel, channel, dispatcher, "left"); err != nil {
			log.Println(err)
			c.JSON(200, NormalRsp{
				Code: 500,
				Msg:  err.Error(),
			})
			cancel()
			return
		}
		portResponse.LeftPort = append(portResponse.LeftPort, ChannelPortPair{
			Channel: channel.Name,
			Port:    port,
		})
	}
	log.Println(j.ChannelInfo.Channels)
	for _, channel := range j.ChannelInfo.Channels {
		port := 0
		if port, err = startListen(ctx, cancel, channel, dispatcher, "right"); err != nil {
			log.Println(err)
			c.JSON(200, NormalRsp{
				Code: 500,
				Msg:  err.Error(),
			})
			cancel()
			return
		}
		portResponse.RightPort = append(portResponse.RightPort, ChannelPortPair{
			Channel: channel.Name,
			Port:    port,
		})
	}
	c.JSON(200, portResponse)
}
func ShutdownHandler(c *gin.Context) {

}
func startListen(ctx context.Context, cancel context.CancelFunc, channel Channel, dispatcher *Dispatcher, side string) (port int, err error) {
	switch channel.Proto {
	case "kcp":
		if port, err = KcpListen(ctx, cancel, channel, dispatcher, side); err != nil {
			log.Println(err)
		}
		return
	default:
		return 0, errors.New("can't match proto ")
	}

}
