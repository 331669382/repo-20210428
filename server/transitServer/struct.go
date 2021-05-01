package main

import "time"

type NormalRsp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}
type Channel struct {
	Name  string `json:"name"`
	Proto string `json:"proto"`
	Param struct {
		Nodelay []int `json:"nodelay"`
		Window  []int `json:"window"`
	} `json:"param"`
	Subscription []string `json:"subscription"`
	OpChannel    bool     `json:"opChannel"`
	FragChannel  bool     `json:"fragChannel"`
}
type RegisterReq struct {
	Token       string `json:"token"`
	ChannelInfo struct {
		OriginList []string  `json:"originList"`
		Channels   []Channel `json:"channels"`
	} `json:"channelInfo"`
}
type ChannelPortPair struct {
	Channel string `json:"channel"`
	Port    int    `json:"port"`
}
type PortResponse struct {
	Code      int               `json:"code"`
	LeftPort  []ChannelPortPair `json:"leftPort"`
	RightPort []ChannelPortPair `json:"rightPort"`
}
type Message []byte //暂时用这个吧
type Dispatcher struct {
	leftInChan   chan Message
	leftOutChan  map[string]chan Message
	rightInChan  chan Message
	rightOutChan map[string]chan Message
}
type MessageConn interface {
	ReadMessage() (msg []byte, err error)
	WriteMessage(msg []byte) error
	SetDeadline(t time.Time) error
	Close() error
}

type ShutDownReq struct{
	
}