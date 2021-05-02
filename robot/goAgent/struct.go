package main

import (
	"time"
)

type Config struct {
	Mode                    string `json:"mode"`
	AgentListenPort         int    `json:"agent_listen_port"`
	DialOrListenChooseTimeS int    `json:"dial_or_listen_choose_time(s)"`
	RemoteoutBlockTimeMs    int    `json:"remoteout_block_time(ms)"`
	RemoteinBlockTimeMs     int    `json:"remotein_block_time(ms)"`
	LocalinBlockTimeMs      int    `json:"localin_block_time(ms)"`
	LocaloutBlockTimeMs     int    `json:"localout_block_time(ms)"`
	KcpNodelay              bool   `json:"kcp_nodelay"`
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

type ConfigStruct struct {
	Role   string `json:"role"`
	Intern struct {
		ServerURI string `json:"serverURI"`
		ClientURI string `json:"clientURI"`
	} `json:"intern"`
	Channels []Channel `json:"channels"`
}
type Dial struct {
	Name  string `json:"name"`
	Proto string `json:"proto"`
	Host  string `json:"host"`
	Port  int    `json:"port"`
}
type Listen struct {
	Name  string `json:"name"`
	Proto string `json:"proto"`
	Host  string `json:"host"`
	Port  int    `json:"port"`
}
type ConnectStruct struct {
	Role           string   `json:"role"`
	Cleanupoldtask int      `json:"cleanupoldtask"`
	Dial           []Dial   `json:"dial"`
	Listen         []Listen `json:"listen"`
	Expired        int64    `json:"expired"`
}
type NormalRspStruct struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type Message []byte //暂时用这个吧

type DispatcherStruct struct {
	remoteInChan  map[string]chan Message //远端进入dispatcher
	remoteOutChan map[string]chan Message //离开dispatcher
	localInChan   chan Message
	localOutChan  chan Message
}
type MessageConn interface {
	ReadMessage() (msg []byte, err error)
	WriteMessage(msg []byte) error
	SetDeadline(t time.Time) error
	Close() error
}
type StatResquest struct {
	Since    int64 `json:"since"`
	Until    int64 `json:"until"`
	Channels []struct {
		Name      string `json:"name"`
		SendBytes int    `json:"sendBytes"`
		RecvBytes int    `json:"recvBytes"`
	} `json:"channels"`
}
type StatResponse struct {
	Code     int    `json:"code"`
	Msg      string `json:"msg,omitempty"`
	Since    int64  `json:"since"`
	Until    int64  `json:"until"`
	Channels []struct {
		Name  string `json:"name"`
		Delay int64  `json:"delay"`
	} `json:"channels"`
}
type EchoMessage struct {
	Op      string `json:"op"`
	Time    int64  `json:"time"`
	Channel string `json:"channel"`
}
