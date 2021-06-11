package common

var RegistryConfig Config

type Config struct {
	Robotlisten     string `json:"robotlisten"`
	Clientlisten    string `json:"clientlisten"`
	Servercertfile  string `json:"serverCertFile"`
	Serverkeyfile   string `json:"serverKeyFile"`
	Clientcafile    string `json:"clientCAFile"`
	Transistservers []struct {
		Name    string `json:"name"`
		Ctlport string `json:"ctlport"`
	} `json:"transistServers"`
	Mode string `json:"mode"` //p2p-只用p2p，其他值两个都用
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
		Status string `json:"status"`
		Expire string `json:"expire"`
		Error  string `json:"error"`
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
		Status string `json:"status"`
		Expire string `json:"expire"`
		Target struct {
			Role string `json:"role"`
			ID   string `json:"id"`
		} `json:"target"`
		Authtype string `json:"authtype"`
	}
}
type PostAuth struct {
	Version string `json:"version"`
	Entity  struct {
		Role string `json:"role"`
		ID   string `json:"id"`
	} `json:"entity"`
	Type    string `json:"type"`
	Trans   string `json:"trans"`
	Time    string `json:"time"`
	Message struct {
		Status string `json:"status"`
		Fin    bool   `json:"fin"`
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
		Error  string      `json:"error"`
		Fin    bool        `json:"fin"`
		Peer   AddressInfo `json:"peer,omitempty"`
		Relay  struct {
			Name    string `json:"name"`
			Ctlport int    `json:"ctlport"`
			Expire  string `json:"expire"`
			Token   string `json:"token"`
			Key     string `json:"key"`
		} `json:"relay,omitempty"`
		Key string `json:"key,omitempty"`
	} `json:"message"`
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

var DefaultChannel []Channel = []Channel{
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
		ID   string `json:"id,omitempty"`
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

type TransitRequest struct {
	Token       string `json:"token"`
	ChannelInfo struct {
		OriginList []string  `json:"originList"`
		Channels   []Channel `json:"channels"`
	} `json:"channelInfo"`
}

type TransitResponse struct {
	Code      int               `json:"code"`
	LeftPort  []ChannelPortPair `json:"leftPort"`
	RightPort []ChannelPortPair `json:"rightPort"`
}
type ChannelPortPair struct {
	Channel string `json:"channel"`
	Port    int    `json:"port"`
}

type RegisterRequest struct {
	Version string `json:"version"`
	Entity  struct {
		Role string `json:"role"`
		ID   string `json:"id"`
	} `json:"entity"`
	Type    string `json:"type"`
	Trans   string `json:"trans"`
	Time    string `json:"time"`
	Message struct {
		Interfaces []struct {
			Name      string `json:"name"`
			Type      string `json:"type"`
			Linkspeed string `json:"linkspeed"`
		} `json:"interfaces"`
		Addresses []struct {
			Type      string `json:"type"`
			Address   string `json:"address"`
			Prefixlen string `json:"prefixlen"`
			Port      int    `json:"port"`
			Interface string `json:"interface"`
			Scope     string `json:"scope"`
			Source    string `json:"source"`
			Priority  string `json:"priority"`
		} `json:"addresses"`
		Gateway []struct {
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
	} `json:"message"`
}
