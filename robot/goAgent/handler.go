package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var CFG ConfigStruct
var dialFinish chan struct{} = make(chan struct{}, 1)
var listenFinish chan struct{} = make(chan struct{}, 1)
var connMapD map[string]interface{} = make(map[string]interface{})
var connMapDlock sync.Mutex
var connMapL map[string]interface{} = make(map[string]interface{})
var delaylistLock sync.Mutex
var delayList map[string]interface{} = make(map[string]interface{})
var flushRate int = 120
var channelSelect chan string = make(chan string, 1)

//存储旧任务的CTX
var allCtx = struct {
	mu      sync.Mutex
	ctxList []struct {
		ctx    context.Context
		cancel context.CancelFunc
	}
}{
	mu: sync.Mutex{},
}
var CTX, CANCEL = context.WithCancel(context.Background())

func ConfigHandler(c *gin.Context) {
	j := ConfigStruct{}
	if err := c.ShouldBindJSON(&j); err != nil {
		Warning.Printf("ConfigHandler:%v\n", err)
		c.JSON(200, NormalRspStruct{
			Code: 500,
			Msg:  err.Error(),
		})
		return
	}
	i := 0
	delayList = make(map[string]interface{})
	for ; i < len(j.Channels); i++ {
		for k := i + 1; k < len(j.Channels); k++ {
			if j.Channels[i].Name == j.Channels[k].Name {
				c.JSON(200, NormalRspStruct{
					Code: 500,
					Msg:  "Channel name must be unique",
				})
				return
			}
		}
		if len(j.Channels[i].Param.Nodelay) != 4 && len(j.Channels[i].Param.Window) != 2 {
			c.JSON(200, NormalRspStruct{
				Code: 500,
				Msg:  "channel.param arg length wrong",
			})
			return
		}
		delayList[j.Channels[i].Name] = make(map[int64]int64)
	}
	if i == 0 {
		c.JSON(200, NormalRspStruct{
			Code: 500,
			Msg:  "channel list is empty",
		})
		return
	}
	CFG = j
	Info.Printf("Accept new CFG:%v\n", CFG)
	c.JSON(200, NormalRspStruct{
		Code: 200,
		Msg:  "set config ok",
	})
}

func TestHandler(c *gin.Context) {
	s := fmt.Sprintf("localin:%d\nlocalout:%d\nremotein:%d\nremoteout:%d\ntopiclist:%v\notherCount:%d\nother:%v\ndelay:%v\n", localInCount, localOutCount, remoteInCount, remoteOutCount, topiclist, otherCount, other, delayList)
	fmt.Println(s)
	c.String(200, s)
}

func ConnectHandler(c *gin.Context) {
	j := ConnectStruct{}
	if err := c.ShouldBindJSON(&j); err != nil {
		Warning.Printf("ConnectHandler:%v\n", err)
		return
	}
	//清除旧任务
	if j.Cleanupoldtask == 1 {
		cleanUpOldTask()
		Info.Println("cleanUpOldTask")
	}

	switch j.Role {
	case "server":
		ctx, _ := addTask()
		dispatcher := newDispatcher(CFG)
		go dispatchWorker(ctx, dispatcher)
		if err := WsConnect(ctx, CFG.Intern.ServerURI, dispatcher); err != nil {
			Error.Printf("Websocket Connect:%v\n", err)
			c.JSON(200, NormalRspStruct{
				Code: 500,
				Msg:  err.Error(),
			})
			cleanUpOldTask()
			Info.Println("cleanUpOldTask")
			return
		}
		var dialErrHost []string
		var listenErrHost []string
		var dialsucc []string
		var listensucc []string
		listenCTX, listenCancel := context.WithCancel(ctx)
		for _, listen := range j.Listen {
			i := 0
			for ; i < len(listenErrHost); i++ {
				if listenErrHost[i] == listen.Host {
					break
				}
			}
			if i != len(listenErrHost) {
				continue
			}
			if err := startListen(listenCTX, listen, dispatcher); err != nil {
				Error.Printf("startListen:%v\n", err)
				listenErrHost = append(listenErrHost, listen.Host)
			}
		}
		for _, listen := range j.Listen {
			i := 0
			for ; i < len(listenErrHost); i++ {
				if listenErrHost[i] == listen.Host {
					break
				}
			}
			if i == len(listenErrHost) {
				listensucc = append(listensucc, listen.Host)
			}
		}
		dialCTX, dialCancel := context.WithCancel(ctx)
		for _, dial := range j.Dial {
			i := 0
			for ; i < len(dialErrHost); i++ {
				if dialErrHost[i] == dial.Host {
					break
				}
			}
			if i != len(dialErrHost) {
				continue
			}
			if err := startDial(dialCTX, dial, dispatcher); err != nil {
				Error.Printf("startDial:%v\n", err)
				dialErrHost = append(dialErrHost, dial.Host)
			}
		}
		if len(connMapD) == len(CFG.Channels) {
			dialFinish <- struct{}{}
		}
		for _, dial := range j.Dial {
			i := 0
			for ; i < len(dialErrHost); i++ {
				if dialErrHost[i] == dial.Host {
					break
				}
			}
			if i == len(dialErrHost) {
				dialsucc = append(dialsucc, dial.Host)
			}
		}
		if len(dialsucc) == 0 && len(listensucc) == 0 {
			c.JSON(200, NormalRspStruct{
				Code: 500,
				Msg:  "Both dial and listen failed",
			})
			cleanUpOldTask()
			Info.Println("cleanUpOldTask")
		} else if len(dialsucc) == 0 {
			channelSelect <- "L"
			dialCancel()
			c.JSON(200, NormalRspStruct{
				Code: 200,
				Msg:  "dial failed,listen succeed",
			})
		} else if len(listensucc) == 0 {
			channelSelect <- "D"
			listenCancel()
			c.JSON(200, NormalRspStruct{
				Code: 200,
				Msg:  "listen failed,dial succeed",
			})
		} else {
			//channel select
			//listen(LAN) first,after dial finish for 2S and listen still not finish,select dial
			go func() {
				select {
				case <-listenFinish:
					Info.Println("Choose Listen channel to use")
					channelSelect <- "L"
					dialCancel()
				case <-dialFinish:
					select {
					case <-listenFinish:
						channelSelect <- "L"
						Info.Println("Choose Listen channel to use")
						dialCancel()
					case <-time.After(time.Second * 2):
						channelSelect <- "D"
						Info.Println("Choose Dial channel to use")
						listenCancel()
					case <-ctx.Done():
						return
					}
				case <-ctx.Done():
					return
				}
			}()
			c.JSON(200, NormalRspStruct{
				Code: 200,
				Msg:  "Both dial and listen succeed,server will chose faster one to use",
			})
		}
		go StratDelayCheck(ctx, flushRate)
	case "client":
		ctx, _ := addTask()
		dispatcher := newDispatcher(CFG)
		go dispatchWorker(ctx, dispatcher)
		if err := WsRunServer(ctx, CFG.Intern.ClientURI, dispatcher); err != nil {
			Error.Printf("Run websocket Server:%v\n", err)
			c.JSON(200, NormalRspStruct{
				Code: 500,
				Msg:  err.Error(),
			})
			cleanUpOldTask()
			Info.Println("cleanUpOldTask")
			return
		}
		var dialErrHost []string
		var dialsucc []string
		for _, dial := range j.Dial {
			i := 0
			for ; i < len(dialErrHost); i++ {
				if dialErrHost[i] == dial.Host {
					break
				}
			}
			if i != len(dialErrHost) {
				continue
			}
			if err := startDial(ctx, dial, dispatcher); err != nil {
				Error.Printf("startDial:%v\n", err)
				dialErrHost = append(dialErrHost, dial.Host)

			}
		}

		for _, dial := range j.Dial {
			i := 0
			for ; i < len(dialErrHost); i++ {
				if dialErrHost[i] == dial.Host {
					break
				}
			}
			if i == len(dialErrHost) {
				dialsucc = append(dialsucc, dial.Host)
			}
		}

		if len(dialsucc) == 0 {
			c.JSON(200, NormalRspStruct{
				Code: 500,
				Msg:  "connection failed",
			})
			return
		} else {
			c.JSON(200, NormalRspStruct{
				Code: 200,
				Msg:  "connection succeed",
			})
		}
		//check stat
		go func() {
			for {
				f := 0
				count := 10
				for _, connList := range connMapD {
					f = 2
					if len(connList.([]*myKcp)) != 1 {
						f = 1
						for i, conn := range connList.([]*myKcp) {
							err := conn.WriteMessage([]byte("ping"))
							if err != nil {
								connList = append(connList.([]*myKcp)[:i], connList.([]*myKcp)[:i+1]...)
							}
						}
						count--
					}
				}
				if count == 0 {
					break
				}
				if f == 0 {
					continue
				}
				if f == 2 {
					break
				}

				time.Sleep(time.Second)
			}
			channelSelect <- "D"
			go StratDelayCheck(ctx, flushRate)
		}()

	default:
		c.JSON(200, NormalRspStruct{
			Code: 500,
			Msg:  "`role` error",
		})
	}

}
func cleanUpOldTask() {
	allCtx.mu.Lock()
	for _, Ctx := range allCtx.ctxList {
		Ctx.cancel()
	}
	allCtx.ctxList = []struct {
		ctx    context.Context
		cancel context.CancelFunc
	}{}
	allCtx.mu.Unlock()
	connMapD = make(map[string]interface{})
	connMapL = make(map[string]interface{})
	dialFinish = make(chan struct{}, 1)
	listenFinish = make(chan struct{}, 1)
	channelSelect = make(chan string, 1)
}
func addTask() (context.Context, context.CancelFunc) {
	var ctx, cancel = context.WithCancel(CTX)
	allCtx.mu.Lock()
	allCtx.ctxList = append(allCtx.ctxList, struct {
		ctx    context.Context
		cancel context.CancelFunc
	}{
		ctx:    ctx,
		cancel: cancel,
	})
	allCtx.mu.Unlock()
	return ctx, cancel
}
func startListen(ctx context.Context, listen Listen, dispatcher *DispatcherStruct) error {
	switch listen.Proto {
	case "kcp":
		if err := KcpListen(ctx, listen, dispatcher); err != nil {
			return err
		}
		return nil

	default:
		return errors.New("can't match proto ")
	}

}
func startDial(ctx context.Context, dial Dial, dispatcher *DispatcherStruct) error {
	switch dial.Proto {
	case "kcp":
		if err := KcpDial(ctx, dial, dispatcher); err != nil {
			return err
		}
		return nil
	default:
		return errors.New("can't match proto ")
	}
}
func StratDelayCheck(ctx context.Context, flush int) {
	DorL := <-channelSelect
	switch DorL {
	case "D":
		for channelName, connList := range connMapD {
			for _, conn := range connList.([]*myKcp) {
				go EchoSendWorker(ctx, channelName, conn)
			}
		}
		go DelayListFlusher(ctx, flush)
	case "L":
		for channelName, connList := range connMapL {
			for _, conn := range connList.([]*myKcp) {
				go EchoSendWorker(ctx, channelName, conn)
			}
		}
		go DelayListFlusher(ctx, flush)
	}
}
func EchoSendWorker(ctx context.Context, channelName string, conn MessageConn) {
	Info.Printf("EchoSendWorker %s start\n", channelName)
	j := EchoMessage{
		Channel: channelName,
		Op:      "echo",
	}
	for {
		select {
		case <-ctx.Done():
			Info.Println("EchoSendWorker killed")
			return
		case <-time.After(time.Millisecond * 500):
			j.Time = time.Now().UnixNano()
			echoMsg, _ := json.Marshal(j)
			if err := conn.WriteMessage(echoMsg); err != nil {
				Info.Printf("EchoSendWorker %s stop\n", channelName)
				return
			}
		}
	}
}
func DelayListFlusher(ctx context.Context, perSecond int) {
	Info.Println("DelayListFlusher start")
	for {
		select {
		case <-ctx.Done():
			Info.Println("DelayListFlusher killed")
			return
		case <-time.After(time.Second * time.Duration(perSecond)):
			for _, delayMap := range delayList {
				timeNow := time.Now().Unix()
				for sendTime := timeNow - int64(perSecond); sendTime <= timeNow; sendTime++ {
					if _, ok := delayMap.(map[int64]int64)[sendTime]; ok {
						delete(delayMap.(map[int64]int64), sendTime)
					}
				}
			}
		}
	}
}
func ShutdownHandler(c *gin.Context) {
	cleanUpOldTask()
	Info.Println("cleanUpOldTask")
	c.String(200, "ok")
}
func StatHandler(c *gin.Context) {
	j := StatResquest{}
	err := c.ShouldBindJSON(&j)
	if err != nil {
		Warning.Printf("StatHandler:%v\n", err)
		c.JSON(200, NormalRspStruct{
			Code: 500,
			Msg:  err.Error(),
		})
	}

	timeNow := time.Now().Unix()
	if timeNow-int64(flushRate) > j.Since {
		j.Since = timeNow - int64(flushRate)
	}
	if timeNow < j.Until {
		j.Until = timeNow
	}
	resp := StatResponse{
		Code:  200,
		Since: j.Since,
		Until: j.Until,
	}
	for channelName, delayMap := range delayList {
		var sum int64
		var count int64
		fmt.Println(delayMap.(map[int64]int64))
		for sendTime := j.Since; sendTime <= j.Until; sendTime++ {
			if _, ok := delayMap.(map[int64]int64)[sendTime]; ok {
				sum += delayMap.(map[int64]int64)[sendTime]
				count += 1
			}
		}
		if count != 0 {
			resp.Channels = append(resp.Channels, struct {
				Name  string `json:"name"`
				Delay int64  `json:"delay"`
			}{
				Name:  channelName,
				Delay: sum / count,
			})
		}
	}
	if len(resp.Channels) == 0 {
		c.JSON(200, NormalRspStruct{
			Code: 500,
			Msg:  "All channels have no delay data for the given time",
		})
		return
	}
	c.JSON(200, resp)
}
func EvenHandler(c *gin.Context) {
	result := tempLog.ReadAndFlush()
	c.JSON(200, struct {
		Code      int      `json:"code"`
		EventList []string `json:"evenList"`
	}{
		Code:      200,
		EventList: result,
	})
}
