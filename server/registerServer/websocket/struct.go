package websocket

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
