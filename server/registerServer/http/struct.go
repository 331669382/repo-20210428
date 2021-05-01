package http

type registerRequest struct {
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
type transitRequest struct {
	Token       string `json:"token"`
	ChannelInfo struct {
		OriginList []string  `json:"originList"`
		Channels   []Channel `json:"channels"`
	} `json:"channelInfo"`
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
type transitResponse struct {
	Code      int               `json:"code"`
	LeftPort  []ChannelPortPair `json:"leftPort"`
	RightPort []ChannelPortPair `json:"rightPort"`
}
type ChannelPortPair struct {
	Channel string `json:"channel"`
	Port    int    `json:"port"`
}
type connectRequest struct {
	Name        string `json:"name"`
	ChannelInfo struct {
		OriginList []string  `json:"originList"`
		Channels   []Channel `json:"channels"`
	} `json:"channelInfo"`
}

var defaultChannel []Channel = []Channel{
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

type connectResponse struct {
	Code        int            `json:"code"`
	ChannelAddr []DialOrListen `json:"channelAddr"`
}
type DialOrListen struct {
	Name  string `json:"name"`
	Proto string `json:"proto"`
	Host  string `json:"host"`
	Port  int    `json:"port"`
}

const (
	connect int = iota
	connectResp
)

type robotConnectRequest struct {
	Op          int `json:"op"`
	ChannelInfo struct {
		OriginList []string  `json:"originList"`
		Channels   []Channel `json:"channels"`
	} `json:"channelInfo"`
	Transit []DialOrListen `json:"transit"`
}
