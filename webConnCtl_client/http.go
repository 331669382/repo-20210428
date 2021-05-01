package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

func connectHandler(c *gin.Context) {
	j := ClientConnReq{}
	err := c.ShouldBindJSON(&j)
	if err != nil {
		e := fmt.Sprintf("get a wrong connect request from client,err:%v", err)
		Error.Println(e)
		c.JSON(200, NormalResp{
			Code: errorCode,
			Msg:  e,
		})
		return
	}
	Info.Printf("receive a connect request from client,target:%s\n", j.Entity.ID)
	err = getAddr()
	if err != nil {
		e := fmt.Sprintf("get client addr err:%v", err)
		Error.Println(e)
		c.JSON(200, NormalResp{
			Code: errorCode,
			Msg:  e,
		})
		return
	}
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{Certificates: []tls.Certificate{_cert}, InsecureSkipVerify: true}}}
	regiReq := RegisterReq{
		Type:    "register",
		Trans:   GetRandomString(6),
		Message: _addressInfo,
	}
	regiReq.Entity.ID = _config.Entity.ID
	regiReq.Entity.Role = _config.Entity.Role
	by, _ := json.Marshal(regiReq)
	Info.Printf("send a register request to registry,addr: %v\n", _addressInfo)
	resp, err := client.Post("https://"+_config.RegisterServer+"/register", "application/json", bytes.NewBuffer(by))
	if err != nil {
		e := fmt.Sprintf("send register request to registry failed:%v", err)
		Error.Println(e)
		c.JSON(200, NormalResp{
			Code: errorCode,
			Msg:  e,
		})
		return
	}
	b, _ := ioutil.ReadAll(resp.Body)
	registerResp := RegisterResp{}
	err = json.Unmarshal(b, &registerResp)
	if err != nil {
		e := fmt.Sprintf("get a wrong register response err:%v, body:%s", err, string(b))
		Error.Println(e)
		c.JSON(200, NormalResp{
			Code: errorCode,
			Msg:  e,
		})
		return
	}
	if registerResp.Message.Status != "success" {
		e := fmt.Sprintf("registry refused the register request,err:%s", registerResp.Message.Error)
		Error.Println(e)
		c.JSON(200, NormalResp{
			Code: errorCode,
			Msg:  e,
		})
		return
	}
	Info.Println("register success,ready to send connect request")
	trans := GetRandomString(6)
	connReq := ConnectReq{
		Type:  "connect",
		Trans: trans,
	}
	connReq.Entity.ID = _config.Entity.ID
	connReq.Entity.Role = _config.Entity.Role
	connReq.Message.Authtype = "preauth"
	connReq.Message.Status = "new"
	connReq.Message.Target.ID = j.Entity.ID
	connReq.Message.Target.Role = "robot"
	by_, _ := json.Marshal(connReq)
	Info.Printf("send a connect request to registry,target:%s\n", connReq.Message.Target.ID)
	resp_, err := client.Post("https://"+_config.RegisterServer+"/connect", "application/json", bytes.NewBuffer(by_))
	if err != nil {
		e := fmt.Sprintf("send connect request to registry failed:%v", err)
		Error.Println(e)
		c.JSON(200, NormalResp{
			Code: errorCode,
			Msg:  e,
		})
		return
	}
	b_, _ := ioutil.ReadAll(resp_.Body)
	connResp := ConnectResp{}
	err = json.Unmarshal(b_, &connResp)
	if err != nil {
		e := fmt.Sprintf("get a wrong connect response err:%v, body:%s", err, string(b_))
		Error.Println(e)
		c.JSON(200, NormalResp{
			Code: errorCode,
			Msg:  e,
		})
		return
	}
	if connResp.Message.Status != "success" {
		e := fmt.Sprintf("registry refused the connect request,err:%s", connResp.Message.Error)
		Error.Println(e)
		c.JSON(200, NormalResp{
			Code: errorCode,
			Msg:  e,
		})
		return
	}
	Info.Printf("get connect response from registry,peer:%v+\n", connResp.Message.Peer)
	defaulfMsg := ConnectResp{
		Version: connResp.Version,
		Type:    "connect-response",
		Trans:   connResp.Trans,
	}
	defaulfMsg.Entity.Role = "client"
	defaulfMsg.Entity.ID = _config.Entity.ID
	agentConfig := AgentConfig{
		Role: "client",
	}
	agentConfig.Intern.ClientURI = j.ClientURI
	agentConfig.Channels = append(agentConfig.Channels, _defaultChannel...)
	confReqbody, _ := json.Marshal(agentConfig)

	confReq, _ := http.NewRequest("POST", "http://"+_config.AgentURI+"/conn/config", bytes.NewBuffer(confReqbody))
	confResp, err := http.DefaultClient.Do(confReq)
	if err != nil {
		e := fmt.Sprintf("send config request to agent err:%v", err)
		Error.Println(e)
		defaulfMsg.Message.Status = "error"
		defaulfMsg.Message.Error = e
		by_, _ := json.Marshal(defaulfMsg)
		_, err_ := client.Post("https://"+_config.RegisterServer+"/connect", "application/json", bytes.NewBuffer(by_))
		if err_ != nil {
			e_ := fmt.Sprintf("send error report to registry failed:%v", err_)
			Error.Println(e_)
			e = fmt.Sprintf("%s and %s", e, e_)
		}
		c.JSON(200, NormalResp{
			Code: errorCode,
			Msg:  e,
		})
		return
	}
	confRespBody, _ := ioutil.ReadAll(confResp.Body)
	confRespJSON := NormalResp{}
	err = json.Unmarshal(confRespBody, &confRespJSON)
	if err != nil {
		e := fmt.Sprintf("recv configResp from agent err:%v,body:%s\n", err, string(confRespBody))
		Error.Println(e)
		defaulfMsg.Message.Status = "error"
		defaulfMsg.Message.Error = e
		by_, _ := json.Marshal(defaulfMsg)
		_, err_ := client.Post("https://"+_config.RegisterServer+"/connect", "application/json", bytes.NewBuffer(by_))
		if err_ != nil {
			e_ := fmt.Sprintf("send error report to registry failed:%v", err_)
			Error.Println(e_)
			e = fmt.Sprintf("%s and %s", e, e_)
		}
		c.JSON(200, NormalResp{
			Code: errorCode,
			Msg:  e,
		})
		return
	}
	if confRespJSON.Code != succCode {
		e := fmt.Sprintf("set agent config error:%s", confRespJSON.Msg)
		Error.Println(e)
		defaulfMsg.Message.Status = "error"
		defaulfMsg.Message.Error = e
		by_, _ := json.Marshal(defaulfMsg)
		_, err_ := client.Post("https://"+_config.RegisterServer+"/connect", "application/json", bytes.NewBuffer(by_))
		if err_ != nil {
			e_ := fmt.Sprintf("send error report to registry failed:%v", err_)
			Error.Println(e_)
			e = fmt.Sprintf("%s and %s", e, e_)
		}
		c.JSON(200, NormalResp{
			Code: errorCode,
			Msg:  e,
		})
		return
	}
	Info.Println("send config to agent success")
	conn := AgentConnect{
		Role:           "client",
		Cleanupoldtask: clean,
	}
	if j.Clean == unClean {
		conn.Cleanupoldtask = unClean
	}
	for _, addr := range connResp.Message.Peer.Addresses {
		for i, c := range _defaultChannel {
			conn.Dial = append(conn.Dial, DialOrListen{
				Name:  c.Name,
				Proto: c.Proto,
				Host:  addr.Address,
				Port:  addr.Port + i,
			})
		}
	}
	agentConnReqBody, _ := json.Marshal(conn)
	Info.Println("send connect request to agent")
	agentConnResp, err := http.DefaultClient.Post("http://"+_config.AgentURI+"/conn/connect", "application/json", bytes.NewBuffer(agentConnReqBody))
	if err != nil {
		e := fmt.Sprintf("send connect request to agent err:%v\n", err)
		Error.Println(e)
		defaulfMsg.Message.Status = "error"
		defaulfMsg.Message.Error = e
		by_, _ := json.Marshal(defaulfMsg)
		_, err_ := client.Post("https://"+_config.RegisterServer+"/connect", "application/json", bytes.NewBuffer(by_))
		if err_ != nil {
			e_ := fmt.Sprintf("send error report to registry failed:%v", err_)
			Error.Println(e_)
			e = fmt.Sprintf("%s and %s", e, e_)
		}
		c.JSON(200, NormalResp{
			Code: errorCode,
			Msg:  e,
		})
		return
	}
	agentConnRespBody, err := ioutil.ReadAll(agentConnResp.Body)
	if err != nil {
		fmt.Println(err)
	}
	agentRespJson := NormalResp{}
	err = json.Unmarshal(agentConnRespBody, &agentRespJson)
	if err != nil {
		e := fmt.Sprintf("recv connectResp from agent err:%v,body:%s\n", err, string(confRespBody))
		Error.Println(e)
		defaulfMsg.Message.Status = "error"
		defaulfMsg.Message.Error = e
		by_, _ := json.Marshal(defaulfMsg)
		_, err_ := client.Post("https://"+_config.RegisterServer+"/connect", "application/json", bytes.NewBuffer(by_))
		if err_ != nil {
			e_ := fmt.Sprintf("send error report to registry failed:%v", err_)
			Error.Println(e_)
			e = fmt.Sprintf("%s and %s", e, e_)
		}
		c.JSON(200, NormalResp{
			Code: errorCode,
			Msg:  e,
		})
		return
	}
	if agentRespJson.Code != succCode {
		e := fmt.Sprintf(" agent connect error:%s", agentRespJson.Msg)
		Error.Println(e)
		defaulfMsg.Message.Status = "error"
		defaulfMsg.Message.Error = e
		by_, _ := json.Marshal(defaulfMsg)
		_, err_ := client.Post("https://"+_config.RegisterServer+"/connect", "application/json", bytes.NewBuffer(by_))
		if err_ != nil {
			e_ := fmt.Sprintf("send error report to registry failed:%v", err_)
			Error.Println(e_)
			e = fmt.Sprintf("%s and %s", e, e_)
		}
		c.JSON(200, NormalResp{
			Code: errorCode,
			Msg:  e,
		})
		return
	}
	Info.Println("agent connect success")
	c.JSON(200, NormalResp{
		Code: 200,
		Msg:  "success",
	})
}
func statusHandlet(c *gin.Context) {

}

func queryHandler(c *gin.Context) {
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{Certificates: []tls.Certificate{_cert}, InsecureSkipVerify: true}}}
	queryReq := QueryReq{
		Type:  "query",
		Trans: GetRandomString(6),
	}
	queryReq.Entity.ID = _config.Entity.ID
	queryReq.Entity.Role = _config.Entity.Role
	by, _ := json.Marshal(queryReq)
	Info.Printf("send a query request to registry,json:%s\n", by)
	resp, err := client.Post("https://"+_config.RegisterServer+"/query", "application/json", bytes.NewBuffer(by))
	if err != nil {
		Info.Println(err)
		return
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Info.Println(err)
		return
	}
	queryResp := QueryResp{}
	err = json.Unmarshal(b, &queryResp)
	if err != nil {
		e := fmt.Sprintf("get a wrong query response err:%v, body:%s", err, string(b))
		Error.Println(e)
		c.JSON(200, NormalResp{
			Code: errorCode,
			Msg:  e,
		})
		return
	}
	if queryResp.Message.Status != "success" {
		e := fmt.Sprintf("registry refused the query request,err:%s", queryResp.Message.Error)
		Error.Println(e)
		c.JSON(200, NormalResp{
			Code: errorCode,
			Msg:  e,
		})
		return
	}
	Info.Printf("get query response from registry: %v\n", queryResp.Message.OnlineList)
	c.JSON(200, struct {
		Code       int      `json:"code"`
		OnlineList []string `json:"onlinelist"`
	}{
		Code:       succCode,
		OnlineList: queryResp.Message.OnlineList,
	})
}
