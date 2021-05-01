package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

var localInCount int = 0
var localOutCount int = 0
var remoteInCount int = 0
var remoteOutCount int = 0
var otherCount int = 0
var topiclist map[string]int = make(map[string]int)
var other []map[string]interface{}

func newDispatcher(CFG ConfigStruct) *DispatcherStruct {
	var dispatcher DispatcherStruct
	dispatcher.localInChan = make(chan Message)
	dispatcher.localOutChan = make(chan Message)
	dispatcher.remoteInChan = make(map[string]chan Message)
	dispatcher.remoteOutChan = make(map[string]chan Message)
	for _, channel := range CFG.Channels {
		dispatcher.remoteOutChan[channel.Name] = make(chan Message)
		dispatcher.remoteInChan[channel.Name] = make(chan Message)
	}
	return &dispatcher
}
func dispatchWorker(ctx context.Context, str *DispatcherStruct) {
	Info.Println("dispatchWorker start")
	go func(ctx context.Context, str *DispatcherStruct) {
		for {
			select {
			case msg := <-str.localInChan:
				msgtype, topic := getMsgType(msg)
				chNames := decideChan(msgtype, topic) //实际使用暂时只会有一个目标频道
				fmt.Println(chNames, topic)
				for _, chName := range chNames {
					select {
					case str.remoteOutChan[chName] <- msg: //这里如果block说明需要丢弃消息，做在这可能方便统计？
						// case <-time.After(time.Millisecond * 100):
						// 	remoteOutCount++
					}

				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx, str)
	for _, remoteIn := range str.remoteInChan {
		go func(ctx context.Context, remoteInChan chan Message) {
			for {
				select {
				case msg := <-remoteInChan:
					select {
					case str.localOutChan <- msg: //考虑下这个block的情况，就是Web UI没连上的情况
						// case <-time.After(time.Millisecond * 500):
						// 	localOutCount++
					}
				case <-ctx.Done():
					return
				}

			}
		}(ctx, remoteIn)
	}
	// go func(ctx context.Context, str *DispatcherStruct) {
	// 	for {
	// 		select {
	// 		case msg := <-str.remoteInChan:
	// 			select {
	// 			case str.localOutChan <- msg: //考虑下这个block的情况，就是Web UI没连上的情况
	// 				// case <-time.After(time.Millisecond * 100):
	// 				// 	localOutCount++
	// 			}
	// 		case <-ctx.Done():
	// 			return
	// 		}

	// 	}
	// }(ctx, str)

}

type echoType int

const (
	notEcho        echoType = 0
	isEcho         echoType = 1
	isEchoResponse echoType = 2
)

func checkEcho(msg Message) (echoType, int64, int64, Message) {
	j := EchoMessage{}
	if err := json.Unmarshal(msg, &j); err != nil {
		Warning.Printf("checkEcho err:%v,msg:%s\n", err, string(msg))
		return notEcho, -1, -1, nil
	}
	if j.Op == "echo" {
		j.Op = "echoresponse"
		newMsg, _ := json.Marshal(j)
		return isEcho, -1, -1, newMsg
	}
	if j.Op == "echoresponse" {
		return isEchoResponse, j.Time / 1e9, time.Now().UnixNano() - j.Time, nil
	}
	// if _, ok := m["op"]; ok {
	// 	if m["op"].(string) == "echo" {
	// 		m["op"] = "echoresponse"
	// 		newMsg, _ := json.Marshal(m)
	// 		return isEcho, -1, -1, newMsg
	// 	}
	// 	if m["op"].(string) == "echoresponse" {
	// 		return isEcho, m["time"].(int64) / 1e9, time.Now().UnixNano() - m["time"].(int64), nil
	// 	}
	// }
	return notEcho, -1, -1, nil
}
func getMsgType(msg Message) (msgtype string, topic string) {
	m := make(map[string]interface{})
	if err := json.Unmarshal(msg, &m); err != nil {
		Warning.Printf("getMsgType:%v\n", err)
		return "", ""
	}
	if _, ok := m["op"]; ok {
		if _, ok := m["topic"]; ok {
			if _, ok := topiclist[m["topic"].(string)]; ok {
				topiclist[m["topic"].(string)]++
			} else {
				topiclist[m["topic"].(string)] = 1
			}
			return "op", m["topic"].(string)
		}
		otherCount++
		other = append(other, m)
		return "op", ""
	}
	otherCount++
	other = append(other, m)
	return "frag", ""
}
func decideChan(msgtype string, topic string) (chNames []string) {
	switch msgtype {
	case "op":
		for _, channel := range CFG.Channels {
			if channel.OpChannel {
				for _, sub := range channel.Subscription {
					if sub == "*" || sub == topic {
						chNames = append(chNames, channel.Name)
						return
					}
				}
			}
		}
	case "frag":
		for _, channel := range CFG.Channels {
			if channel.FragChannel {
				for _, sub := range channel.Subscription {
					if sub == "*" || sub == topic {
						chNames = append(chNames, channel.Name)
						return
					}
				}
			}
		}
	}
	return
}
func remoteWriteHandleFunc(ctx context.Context, conn MessageConn, remoteOutChan <-chan Message, channelName string) {
	Info.Printf("%s remoteWrite start\n", channelName)
	for {
		select {
		case msg := <-remoteOutChan:
			err := conn.WriteMessage(msg)
			if err != nil {
				Error.Printf("remoteWrite:%v\n", err)
				return
			}
		case <-ctx.Done():
			conn.WriteMessage([]byte("closed"))
			conn.Close()
			Info.Printf("%s remoteWrite stop\n", channelName)
			return
		}
	}
}

func remoteReadHandleFunc(ctx context.Context, conn MessageConn, remoteInChan chan<- Message, channelName string) {
	Info.Printf("%s remoteRead start\n", channelName)
	for {
		msg, err := conn.ReadMessage()
		if err != nil {
			Error.Printf("remoteRead:%v\n", err)
			return
		}
		if string(msg) == "closed" {
			conn.Close()
			Info.Printf("channel [%s]Read closed msg\n", channelName)
			return
		}
		switch eType, sendTime, delay, responseMsg := checkEcho(msg); eType {
		case isEcho:
			err := conn.WriteMessage(responseMsg)
			if err != nil {
				Warning.Printf("write echoresponse error:%v\n", err)
			}
		case isEchoResponse:
			delaylistLock.Lock()
			if _, ok := delayList[channelName].(map[int64]int64)[sendTime]; ok {
				delayList[channelName].(map[int64]int64)[sendTime] = (delay + delayList[channelName].(map[int64]int64)[sendTime]) / 2
			} else {
				delayList[channelName].(map[int64]int64)[sendTime] = delay
			}
			delaylistLock.Unlock()
		case notEcho:
			select {
			case remoteInChan <- msg:
			// case <-time.After(time.Millisecond * 100):
			// 	remoteInCount++
			case <-ctx.Done():
				Info.Printf("%s remoteRead stop\n", channelName)
				break
			}
		}
	}

}

func localReadHandleFunc(ctx context.Context, ws *websocket.Conn, localInChan chan<- Message) {
	Info.Println("localReadHandleFunc start")
	for {

		_, msg, err := ws.ReadMessage()
		if err != nil {
			Error.Printf("localRead error:%v\n", err)
			return
		}

		select {
		case localInChan <- msg:
		// case <-time.After(time.Millisecond * 100):
		// 	localInCount++
		case <-ctx.Done():
			Info.Println("localReadHandleFunc stop")
			break
		}
	}

}
func localWriteHandleFunc(ctx context.Context, ws *websocket.Conn, localOutChan <-chan Message) {
	Info.Println("localWriteHandleFunc start")
	for {
		select {
		case msg := <-localOutChan:
			err := ws.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				Error.Printf("localWrite:%v\n", err)
				return
			}
		case <-ctx.Done():
			Info.Println("localWriteHandleFunc stop")
			return
		}
	}
}
