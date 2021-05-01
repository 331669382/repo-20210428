package main

const (
	unClean int = iota
	clean
)
const (
	succCode  int = 200
	errorCode int = 500
)

type AgentConfig struct {
	Role   string `json:"role"`
	Intern struct {
		ServerURI string `json:"serverURI"`
		ClientURI string `json:"clientURI"`
	} `json:"intern"`
	Channels []Channel `json:"channels"`
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

type AgentConnect struct {
	Role           string         `json:"role"`
	Cleanupoldtask int            `json:"cleanupoldtask"`
	Dial           []DialOrListen `json:"dial"`
	Listen         []DialOrListen `json:"listen"`
	Expired        int64          `json:"expired"`
}
type DialOrListen struct {
	Name  string `json:"name"`
	Proto string `json:"proto"`
	Host  string `json:"host"`
	Port  int    `json:"port"`
}

type AddressInfo struct {
	Interfaces []struct {
		Name      string `json:"name"`
		Type      string `json:"type"`
		Linkspeed string `json:"linkspeed"`
	} `json:"interfaces"`
	Addresses []Address `json:"addresses"`
	Gateway   []struct {
		Type      string `json:"type"`
		Address   string `json:"address"`
		Interface string `json:"interface"`
		Priority  string `json:"priority"`
	} `json:"gateway"`
	Domain []struct {
		Type string `json:"type"`
		Name string `json:"name"`
		Port string `json:"port"`
	} `json:"domain"`
}
type Address struct {
	Type      string `json:"type"`
	Address   string `json:"address"`
	Prefixlen string `json:"prefixlen"`
	Port      int    `json:"port"`
	Interface string `json:"interface"`
	Scope     string `json:"scope"`
	Source    string `json:"source"`
	Priority  string `json:"priority"`
}

type Config struct {
	RegisterServer string `json:"registerserver"`
	AgentURI       string `json:"agenturi"`
	AgentBasePort  int    `json:"agentbaseport"`
	Entity         struct {
		Role        string `json:"role"`
		ID          string `json:"id"`
		Certificate string `json:"certificate"`
		Key         string `json:"key"`
	} `json:"entity"`
	Authorizedclients []string `json:"authorizedclients"`
	Adminport         string   `json:"adminport"`
}

type QueryReq struct {
	Version string `json:"version"`
	Entity  struct {
		Role string `json:"role"`
		ID   string `json:"id"`
	} `json:"entity"`
	Type  string `json:"type"`
	Trans string `json:"trans"`
	Time  string `json:"time"`
}
type QueryResp struct {
	Version string `json:"version"`
	Entity  struct {
		Role string `json:"role"`
		ID   string `json:"id"`
	} `json:"entity"`
	Type    string `json:"type"`
	Trans   string `json:"trans"`
	Time    string `json:"time"`
	Message struct {
		Status     string   `json:"status,omitempty"`
		Error      string   `json:"error,omitempty"`
		OnlineList []string `json:"onlinelist"`
	} `json:"message"`
}
type RegisterReq struct {
	Version string `json:"version"`
	Entity  struct {
		Role string `json:"role"`
		ID   string `json:"id"`
	} `json:"entity"`
	Type    string      `json:"type"`
	Trans   string      `json:"trans"`
	Time    string      `json:"time"`
	Message AddressInfo `json:"message"`
}
type RegisterResp struct {
	Version string `json:"version"`
	Entity  struct {
		Role string `json:"role"`
		ID   string `json:"id"`
	} `json:"entity"`
	Type    string `json:"type"`
	Trans   string `json:"trans"`
	Time    string `json:"time"`
	Message struct {
		Status string `json:"status,omitempty"`
		Expire string `json:"expire"`
		Error  string `json:"error,omitempty"`
	} `json:"message"`
}

type ConnectReq struct {
	Version string `json:"version"`
	Entity  struct {
		Role string `json:"role"`
		ID   string `json:"id"`
	} `json:"entity"`
	Type    string `json:"type"`
	Trans   string `json:"trans"`
	Time    string `json:"time"`
	Message struct {
		Status string `json:"status,omitempty"`
		Expire string `json:"expire"`
		Target struct {
			Role string `json:"role"`
			ID   string `json:"id"`
		} `json:"target"`
		Authtype string `json:"authtype"`
	}
	Transit struct {
		Host string `json:"host,omitempty"`
		Port []int  `json:"port,omitempty"`
	}
}
type ConnectResp struct {
	Version string `json:"version"`
	Entity  struct {
		Role string `json:"role"`
		ID   string `json:"id"`
	} `json:"entity"`
	Type    string `json:"type"`
	Trans   string `json:"trans"`
	Time    string `json:"time"`
	Message struct {
		Status string      `json:"status"`
		Error  string      `json:"error,omitempty"`
		Fin    bool        `json:"fin"`
		Peer   AddressInfo `json:"peer,omitempty"`
		Relay  struct {
			Name    string `json:"name"`
			Ctlport string `json:"ctlport"`
			Expire  string `json:"expire"`
			Token   string `json:"token"`
			Key     string `json:"key"`
		} `json:"relay,omitempty"`
		Key string `json:"key,omitempty"`
	} `json:"message"`
}

var _defaultChannel []Channel = []Channel{
	{
		Name:  "cmd_vel",
		Proto: "kcp",
		Param: struct {
			Nodelay []int "json:\"nodelay\""
			Window  []int "json:\"window\""
		}{
			Nodelay: []int{1, 20, 2, 1},
			Window:  []int{0, 0},
		},
		Subscription: []string{"/cmd_vel"},
		OpChannel:    true,
		FragChannel:  true,
	},
	{
		Name:  "critical",
		Proto: "kcp",
		Param: struct {
			Nodelay []int "json:\"nodelay\""
			Window  []int "json:\"window\""
		}{
			Nodelay: []int{1, 20, 2, 1},
			Window:  []int{0, 0},
		},
		Subscription: []string{"*"},
		OpChannel:    true,
		FragChannel:  true,
	},
}

type ClientConnReq struct {
	Entity struct {
		Role string `json:"role"`
		ID   string `json:"id"`
	} `json:"entity"`
	Clean     int    `json:"clean"`
	ClientURI string `json:"clientURI"`
}

type NormalResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}
