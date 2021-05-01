package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"mcurobot.com/registerServer/common"
	"mcurobot.com/registerServer/log"
	"mcurobot.com/registerServer/websocket"
)

var TransitServer string = "47.112.96.50"	// 在registerServer/main.go覆盖此数值
var TransitPort string = "45001"	// 在registerServer/main.go覆盖此数值
var OnlineClient map[string]common.AddressInfo = make(map[string]common.AddressInfo)

func registerHandler(c *gin.Context) {
	resp := common.RegisterResp{
		Type: "register-response",
	}
	resp.Entity.Role = "registry"
	if c.Request.TLS == nil || c.Request.TLS.PeerCertificates == nil || c.Request.TLS.PeerCertificates[0].Subject.CommonName == "" {
		body := make([]byte, 1024)
		n, _ := c.Request.Body.Read(body)
		e := fmt.Sprintf("unkown client send query request with invalid tls common name,body:%s", body[:n])
		resp.Message.Status = "error"
		resp.Message.Error = e
		log.Error.Println(e)
		c.JSON(200, resp)
		return
	}
	j := common.RegisterReq{}
	err := c.ShouldBindJSON(&j)
	if err != nil {
		body := make([]byte, 1024)
		n, _ := c.Request.Body.Read(body)
		e := fmt.Sprintf("receive a invalid register json from client,err:%s,body:%s", err, string(body[:n]))
		resp.Message.Status = "error"
		resp.Message.Error = e
		log.Error.Println(e)
		c.JSON(200, resp)
		return
	}
	if j.Entity.ID != c.Request.TLS.PeerCertificates[0].Subject.CommonName {
		body := make([]byte, 1024)
		n, _ := c.Request.Body.Read(body)
		e := fmt.Sprintf("client's common name is not equal entity id,body:%s", body[:n])
		resp.Message.Status = "error"
		resp.Message.Error = e
		log.Error.Println(e)
		c.JSON(200, resp)
		return
	}
	clientName := j.Entity.ID
	log.Info.Printf("receive a register request from client [%s]\n", clientName)
	OnlineClient[clientName] = j.Message
	resp.Version = j.Version
	resp.Trans = j.Trans
	resp.Message.Status = "success"
	c.JSON(200, resp)
}
func queryHandler(c *gin.Context) {
	j := common.QueryReq{}
	resp := common.QueryResp{
		Type: "query-response",
	}
	resp.Entity.Role = "registry"
	// if c.Request.TLS == nil || c.Request.TLS.PeerCertificates == nil || c.Request.TLS.PeerCertificates[0].Subject.CommonName == "" {
	// 	body := make([]byte, 1024)
	// 	n, _ := c.Request.Body.Read(body)
	// 	e := fmt.Sprintf("unkown client send connect request with invalid tls common name,body:%s", body[:n])
	// 	resp.Message.Status = "error"
	// 	resp.Message.Error = e
	// 	log.Error.Println(e)
	// 	c.JSON(200, resp)
	// 	return
	// }

	err := c.ShouldBindJSON(&j)
	if err != nil {
		body := make([]byte, 1024)
		n, _ := c.Request.Body.Read(body)
		e := fmt.Sprintf("receive a invalid query json from client,err:%s,body:%s", err, string(body[:n]))
		resp.Message.Status = "error"
		resp.Message.Error = e
		log.Error.Println(e)
		c.JSON(200, resp)
		return
	}
	clientName := j.Entity.ID
	log.Info.Printf("receive a query request from client [%s]\n", clientName)
	onlineList := []string{}
	for name := range websocket.OnlineRobotWs {
		onlineList = append(onlineList, name)
	}
	resp.Version = j.Version
	resp.Trans = j.Trans
	resp.Message.Status = "success"
	resp.Message.OnlineList = onlineList
	c.JSON(200, resp)
}
func connectHandler(c *gin.Context) {
	body := make([]byte, 2048)
	n, _ := c.Request.Body.Read(body)
	resp := common.ConnectResp{
		Type: "connect-response",
	}
	resp.Entity.Role = "registry"
	if c.Request.TLS == nil || c.Request.TLS.PeerCertificates == nil || c.Request.TLS.PeerCertificates[0].Subject.CommonName == "" {
		body := make([]byte, 1024)
		n, _ := c.Request.Body.Read(body)
		e := fmt.Sprintf("unkown client send query request with invalid tls common name,body:%s", body[:n])
		resp.Message.Status = "error"
		resp.Message.Error = e
		log.Error.Println(e)
		c.JSON(200, resp)
		return
	}
	j := common.ConnectReq{}
	err := json.Unmarshal(body[:n], &j)
	if err != nil {
		e := fmt.Sprintf("receive a invalid connect json from client,err:%s", err)
		resp.Message.Status = "error"
		resp.Message.Error = e
		c.JSON(200, resp)
		log.Error.Printf("%s,body:%s\n", e, body)
		return
	}
	if j.Entity.ID != c.Request.TLS.PeerCertificates[0].Subject.CommonName {
		body := make([]byte, 1024)
		n, _ := c.Request.Body.Read(body)
		e := fmt.Sprintf("client's common name is not equal entity id,body:%s", body[:n])
		resp.Message.Status = "error"
		resp.Message.Error = e
		log.Error.Println(e)
		c.JSON(200, resp)
		return
	}
	clientName := j.Entity.ID
	log.Info.Printf("receive a connect request from client [%s]\n", clientName)
	if _, ok := OnlineClient[clientName]; !ok {
		e := fmt.Sprintf("refused a connect request from unregister client [%s]", clientName)
		resp.Message.Status = "error"
		resp.Message.Error = e
		c.JSON(200, resp)
		log.Error.Printf("%s,body:%s\n", e, body)
		return
	}
	resp.Trans = j.Trans
	if _, ok := websocket.OnlineRobotWs[j.Message.Target.ID]; ok {
		if _, ok_ := websocket.RobotInfo[j.Message.Target.ID]; ok_ {
			transitReqStruct := common.TransitRequest{
				Token: "123",
			}
			transitReqStruct.ChannelInfo.Channels = common.DefaultChannel
			js, _ := json.Marshal(transitReqStruct)
			transitReq, err := http.NewRequest("POST", "http://"+TransitServer+":"+TransitPort+"/register", bytes.NewBuffer(js))
			if err != nil {
				e := fmt.Sprintf("New register Request to transitServer err:%v", err)
				resp.Message.Status = "error"
				resp.Message.Error = e
				c.JSON(200, resp)
				log.Error.Println(e)
				return
			}
			transitResp, err := http.DefaultClient.Do(transitReq)
			if err != nil {
				e := fmt.Sprintf("Do register Request to transitServer err:%v", err)
				resp.Message.Status = "error"
				resp.Message.Error = e
				c.JSON(200, resp)
				log.Error.Println(e)
				return
			}
			respStruct := common.TransitResponse{}
			transitRespBody, err := ioutil.ReadAll(transitResp.Body)
			if err != nil {
				e := fmt.Sprintf("Read transitSever register Response from transitServer err:%v", err)
				resp.Message.Status = "error"
				resp.Message.Error = e
				c.JSON(200, resp)
				log.Error.Println(e)
				return
			}
			err = json.Unmarshal(transitRespBody, &respStruct)
			if err != nil {
				e := fmt.Sprintf("Read transitSever register Response json from transitServer err:%v", err)
				resp.Message.Status = "error"
				resp.Message.Error = e
				c.JSON(200, resp)
				log.Error.Printf("%s,body:%s\n", e, string(transitRespBody))
				return
			}
			robotConnReq := common.ConnectReq{
				Version: j.Version,
				Type:    "connect",
				Trans:   j.Trans,
			}
			for i, channelInfo := range transitReqStruct.ChannelInfo.Channels {
				robotConnReq.Transit.Host = TransitServer
				robotConnReq.Transit.Port = append(robotConnReq.Transit.Port, respStruct.LeftPort[i].Port)
				if channelInfo.Name != respStruct.LeftPort[i].Channel || channelInfo.Name != respStruct.RightPort[i].Channel {
					e := "Read transitSever register Response err: response channel wrong"
					resp.Message.Status = "error"
					resp.Message.Error = e
					c.JSON(200, resp)
					log.Error.Printf("%s,body:%s\n", e, string(transitRespBody))
					return
				}
			}
			if respStruct.LeftPort == nil || respStruct.RightPort == nil {
				e := "Read register Response err: respStruct LeftPort or RightPort is nil"
				resp.Message.Status = "error"
				resp.Message.Error = e
				c.JSON(200, resp)
				log.Error.Printf("%s,body:%s\n", e, string(transitRespBody))
				return
			}
			robotConnReq.Entity.Role = "registry"
			robotConnReq.Entity.ID = clientName
			robotConnReq.Message.Status = "new"
			robotConnReq.Message.Target.ID = j.Message.Target.ID
			robotConnReq.Message.Target.Role = "robot"
			err = websocket.OnlineRobotWs[j.Message.Target.ID].WriteJSON(robotConnReq)
			if err != nil {
				e := fmt.Sprintf("send connect request to robot err:%v", err)
				resp.Message.Status = "error"
				resp.Message.Error = e
				c.JSON(200, resp)
				log.Error.Println(e)
				websocket.OnlineRobotWs[j.Message.Target.ID].Close()
				return
			}
			robotConnResp := common.ConnectResp{}
			err = websocket.OnlineRobotWs[j.Message.Target.ID].ReadJSON(&robotConnResp)
			if err != nil {
				e := fmt.Sprintf("read connect response from robot err:%v", err)
				resp.Message.Status = "error"
				resp.Message.Error = e
				c.JSON(200, resp)
				log.Error.Println(e)
				websocket.OnlineRobotWs[j.Message.Target.ID].Close()
				return
			}
			if robotConnResp.Message.Status != "pending" {
				e := fmt.Sprintf("robot refused the connection,err:%s", robotConnResp.Message.Error)
				resp.Message.Status = "error"
				resp.Message.Error = e
				c.JSON(200, resp)
				log.Error.Println(e)
				return
			}
			resp.Message.Status = "success"
			peer := websocket.RobotInfo[j.Message.Target.ID]
			peer.Addresses = append(peer.Addresses, common.Address{
				Address: TransitServer,
				Port:    respStruct.RightPort[0].Port,
			})
			resp.Message.Peer = peer
			log.Info.Printf("client [%s] connect request to robot [%s] success\n", clientName, j.Message.Target.ID)
			c.JSON(200, resp)
			return
		}
	}
	e := fmt.Sprintf("target robot of connect request offline")
	resp.Message.Status = "error"
	resp.Message.Error = e
	c.JSON(200, resp)
	log.Error.Printf("%s,body:%s\n", e, body)
}