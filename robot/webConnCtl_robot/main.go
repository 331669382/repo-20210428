package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prestonTao/upnp"
)

var _rosbridgeURI string = "ws://0.0.0.0:9090"
var _addressInfo AddressInfo
var _config Config
var _state string

func main() {
	file, err := os.Open("config.config")
	if err != nil {
		fmt.Printf("Open ./config.config failed [Err:%v]", err)
		return
	}

	err = json.NewDecoder(file).Decode(&_config)
	if err != nil {
		fmt.Println("invalid config file")
		return
	}
	err = SetLog()
	if err != nil {
		fmt.Println(err)
		return
	}
	mapping := new(upnp.Upnp)
	for i := range _defaultChannel {
		if err := mapping.AddPortMapping(_config.AgentBasePort+i, _config.AgentBasePort+i, "UDP"); err == nil {
			Info.Printf("AddPortMapping %d success\n", _config.AgentBasePort+i)
			mapping.Reclaim()
		} else {
			Error.Printf("AddPortMapping %d fail\n", _config.AgentBasePort+i)
		}
	}

	cert, err := tls.X509KeyPair([]byte(_config.Entity.Certificate), []byte(_config.Entity.Key))
	if err != nil {
		fmt.Printf("load cert err:%v", err)
		return
	}
	dialer := websocket.Dialer{TLSClientConfig: &tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true}}
	if _config.Entity.Role == "robot" {
		for {
			_state = "boot"
			Info.Println("State: boot")
			err := getAddr()
			if err != nil {
				Error.Println(err)
				return
			}
			ws, _, err := dialer.Dial(fmt.Sprintf("wss://%s", _config.RegisterServer), nil)
			if err != nil {
				Error.Printf("Dial error:err%v\n", err)
				time.Sleep(time.Second * 5)
				continue
			}
			Info.Println("Dial registry succ")
			regiReq := RegisterReq{
				Type:    "register",
				Trans:   GetRandomString(6),
				Message: _addressInfo,
			}
			regiReq.Entity.ID = _config.Entity.ID
			regiReq.Entity.Role = _config.Entity.Role
			err = ws.WriteJSON(regiReq)
			if err != nil {
				Error.Printf("send robotConfig to registy err:%v, try to reconnect\n", err)
				continue
			}
			regiResp := RegisterResp{}
			err = ws.ReadJSON(&regiResp)
			if err != nil {
				Error.Printf("json unmarshal register response err:%v\n", err)
				continue
			}
			if regiResp.Message.Status != "success" {
				Error.Printf("registy refused the register request, the response is:%v+\n", regiResp)
				continue
			}
			ctx, cancel := context.WithCancel(context.Background())
			go func(context.Context) {
				for {
					select {
					case <-time.After(time.Second * 60):
						err := getAddr()
						if err != nil {
							Error.Println(err)
							continue
						}
						regiReq := RegisterReq{
							Type:    "register",
							Trans:   GetRandomString(6),
							Message: _addressInfo,
						}
						regiReq.Entity.ID = _config.Entity.ID
						regiReq.Entity.Role = _config.Entity.Role
						err = ws.WriteJSON(regiReq)
						if err != nil {
							Error.Printf("send robotConfig to registy err:%v, try to reconnect\n", err)
							return
						}
						Info.Println("send register request successfully")
					case <-ctx.Done():
						return
					}
				}
			}(ctx)
			_state = "ready"
			Info.Println("register success,State: ready")
			for {
				msgType, msg, err := ws.ReadMessage()
				if err != nil {
					Error.Printf("ReadMessage err:%v, try to reconnect\n", err)
					break
				}
				if msgType == websocket.PingMessage {
					continue
				}
				if msgType == websocket.TextMessage {
					m := make(map[string]interface{})
					err := json.Unmarshal(msg, &m)
					if err != nil || m["type"] == nil || m["trans"] == nil {
						Error.Printf("read json msg err:%v, msg:%s\n", err, string(msg))
						err := ws.WriteMessage(websocket.TextMessage, []byte("{\"error\":\"wrong json\"}"))
						if err != nil {
							Error.Printf("send Response to registy err:%v, try to reconnect\n", err)
							break
						}
						continue
					}
					switch m["type"].(string) {
					case "connect-request":
						connectRequest := ConnectReq{}
						err := json.Unmarshal(msg, &connectRequest)
						if err != nil {
							Error.Printf("read json msg with a wrong connectRequest:%v, msg:%s\n", err, string(msg))
							continue
						}
						switch connectRequest.Message.Status {
						case "new":
							var i int
							postAuth := PostAuth{
								Version: connectRequest.Version,
								Type:    "connect-postauth",
								Trans:   connectRequest.Trans,
							}
							for _, authClient := range _config.Authorizedclients {
								if authClient == connectRequest.Message.Target.ID {
									break
								}
								i++
							}
							if i == len(_config.Authorizedclients) {
								Info.Printf("receive a connect request from unauthorizedclient %s\n", connectRequest.Message.Target.ID)
								postAuth.Message.Fin = true
								postAuth.Message.Status = "failed"
								err := ws.WriteJSON(postAuth)
								if err != nil {
									Error.Printf("send postAuth to registy err:%v, try to reconnect\n", err)
									break
								}
								continue
							}
							Info.Printf("receive a connect request from authorizedclient %s\n", connectRequest.Message.Target.ID)
							postAuth.Message.Fin = false
							postAuth.Message.Status = "success"
							err := ws.WriteJSON(postAuth)
							if err != nil {
								Error.Printf("send postAuth to registy err:%v, try to reconnect\n", err)
								break
							}
						case "preauth":
						case "established":
						}
					case "connect-response":
						connectResp := ConnectResp{}
						err := json.Unmarshal(msg, &connectResp)
						if err != nil {
							Error.Printf("read json msg with a wrong connectRequest:%v, msg:%s\n", err, string(msg))
							continue
						}
						defaulfMsg := ConnectResp{
							Version: connectResp.Version,
							Type:    "connect-response",
							Time:    fmt.Sprintf("%d", time.Now().Unix()),
							Trans:   connectResp.Trans,
						}
						defaulfMsg.Entity.Role = "robot"
						defaulfMsg.Entity.ID = _config.Entity.ID
						defaulfMsg.Message.Fin = true
						switch connectResp.Message.Status {
						case "pending":
							_state = "pending"
							Info.Printf("get a connectResponse from registry,State: pending\n")
							agentConfig := AgentConfig{
								Role: "server",
							}
							agentConfig.Intern.ServerURI = _rosbridgeURI
							agentConfig.Channels = append(agentConfig.Channels, _defaultChannel...)
							confReqbody, _ := json.Marshal(agentConfig)
							confReq, _ := http.NewRequest("POST", "http://"+_config.AgentURI+"/conn/config", bytes.NewBuffer(confReqbody))
							confResp, err := http.DefaultClient.Do(confReq)
							if err != nil {
								Error.Printf("send config request to agent err:%v\n", err)
								defaulfMsg.Message.Status = "error"
								defaulfMsg.Message.Error = fmt.Sprintf("send config request to agent err:%v", err)
								err := ws.WriteJSON(defaulfMsg)
								if err != nil {
									Error.Printf("send connectResponse to registy err:%v, try to reconnect\n", err)
									break
								}
								_state = "ready"
								continue
							}
							confRespBody, _ := ioutil.ReadAll(confResp.Body)
							confResp.Body.Close()
							confRespJSON := NormalResp{}
							err = json.Unmarshal(confRespBody, &confRespJSON)
							if err != nil {
								Error.Printf("recv config Resp from agent err:%v,body:%s\n", err, confRespBody)
								defaulfMsg.Message.Status = "error"
								defaulfMsg.Message.Error = fmt.Sprintf("recv config Resp from agent err:%v,body:%s", err, confRespBody)
								err := ws.WriteJSON(defaulfMsg)
								if err != nil {
									Error.Printf("send connectResponse to registy err:%v, try to reconnect\n", err)
									break
								}
								_state = "ready"
								continue
							}
							if confRespJSON.Code != succCode {
								defaulfMsg.Message.Status = "error"
								Error.Printf("set agent config error:%s\n", confRespJSON.Msg)
								defaulfMsg.Message.Error = fmt.Sprintf("set agent config error:%s", confRespJSON.Msg)
								err := ws.WriteJSON(defaulfMsg)
								if err != nil {
									Error.Printf("send connectResponse to registy err:%v, try to reconnect\n", err)
									break
								}
								_state = "ready"
								continue
							}
							p2pConn := AgentConnect{
								Role:           "server",
								Cleanupoldtask: clean,
							}
							for _, addr := range connectResp.Message.Peer.Addresses {
								for i, c := range _defaultChannel {
									p2pConn.Dial = append(p2pConn.Dial, DialOrListen{
										Name:  c.Name,
										Proto: c.Proto,
										Host:  addr.Address,
										Port:  addr.Port + i,
									})
								}
							}
							for i, c := range _defaultChannel {
								p2pConn.Listen = append(p2pConn.Listen, DialOrListen{
									Name:  c.Name,
									Proto: c.Proto,
									Host:  "0.0.0.0",
									Port:  _config.AgentBasePort + i,
								})
							}
							p2pConnReqBody, _ := json.Marshal(p2pConn)
							p2pConnReq, _ := http.NewRequest("POST", "http://"+_config.AgentURI+"/conn/connect", bytes.NewBuffer(p2pConnReqBody))
							p2pConnResp, err := http.DefaultClient.Do(p2pConnReq)
							p2pWaitTime := _config.WaitP2PConnectS
							if err != nil {
								Error.Printf("send p2p connect request to agent err:%v,try to connect transitServer\n", err)
								p2pWaitTime = 0
							} else {
								p2pConnRespBody, _ := ioutil.ReadAll(p2pConnResp.Body)
								p2pConnResp.Body.Close()
								p2pConnRespJSON := NormalResp{}
								err = json.Unmarshal(p2pConnRespBody, &p2pConnRespJSON)
								if err != nil {
									Error.Printf("recv p2p connectResp from agent err:%v,body:%s,try to connect transitServer\n", err, p2pConnRespBody)
									p2pWaitTime = 0
								} else {
									if p2pConnRespJSON.Code != succCode {
										Error.Printf("agent p2p connect error %s,try to connect transitServer\n", p2pConnRespJSON.Msg)
										p2pWaitTime = 0
									}
								}

							}
							time.Sleep(time.Second * time.Duration(p2pWaitTime))
							statuResp, err := http.Get("http://" + _config.AgentURI + "statu")
							if err != nil {
								Error.Printf("get agent statu Resp err:%v\n", err)
								defaulfMsg.Message.Status = "error"
								defaulfMsg.Message.Error = fmt.Sprintf("get agent statu Resp err:%v", err)
								err := ws.WriteJSON(defaulfMsg)
								if err != nil {
									Error.Printf("send connectResponse to registy err:%v, try to reconnect\n", err)
									break
								}
								_state = "ready"
								continue
							}
							statuRespBody, _ := ioutil.ReadAll(statuResp.Body)
							statuResp.Body.Close()
							if string(statuRespBody) != "success" {
								Info.Println("p2p mode connect filed,try to connect transit server")
								transitConn := AgentConnect{
									Role:           "server",
									Cleanupoldtask: clean,
								}
								for i, c := range _defaultChannel {
									transitConn.Dial = append(transitConn.Dial, DialOrListen{
										Name:  c.Name,
										Proto: c.Proto,
										Host:  connectResp.Message.Relay.Name,
										Port:  connectResp.Message.Relay.Ctlport + i,
									})
								}
								transitConnReqBody, _ := json.Marshal(transitConn)
								tranConnReq, _ := http.NewRequest("POST", "http://"+_config.AgentURI+"/conn/connect", bytes.NewBuffer(transitConnReqBody))
								transitConnResp, err := http.DefaultClient.Do(tranConnReq)
								if err != nil {
									Error.Printf("send transit connect request to agent err:%v\n", err)
									defaulfMsg.Message.Status = "error"
									defaulfMsg.Message.Error = fmt.Sprintf("send transit connect request to agent err:%v", err)
									err := ws.WriteJSON(defaulfMsg)
									if err != nil {
										Error.Printf("send connectResponse to registy err:%v, try to reconnect\n", err)
										break
									}
									_state = "ready"
									continue
								}
								transitConnRespBody, _ := ioutil.ReadAll(transitConnResp.Body)
								transitConnResp.Body.Close()
								transitConnRespJSON := NormalResp{}
								json.Unmarshal(transitConnRespBody, &transitConnRespJSON)
								if transitConnRespJSON.Code != succCode {
									defaulfMsg.Message.Status = "error"
									Error.Printf("agent connect error %s\n", transitConnRespJSON.Msg)
									defaulfMsg.Message.Error = fmt.Sprintf("agent connect error %s", transitConnRespJSON.Msg)
									err := ws.WriteJSON(defaulfMsg)
									if err != nil {
										Error.Printf("send connectResponse to registy err:%v, try to reconnect\n", err)
										break
									}
									_state = "ready"
									continue
								}
								time.Sleep(time.Second * time.Duration(_config.WaitTransitConnectS))
								statuResp, err := http.Get("http://" + _config.AgentURI + "statu")
								if err != nil {
									Error.Printf("get agent statu Resp err:%v\n", err)
									defaulfMsg.Message.Status = "error"
									defaulfMsg.Message.Error = fmt.Sprintf("get agent statu Resp err:%v", err)
									err := ws.WriteJSON(defaulfMsg)
									if err != nil {
										Error.Printf("send connectResponse to registy err:%v, try to reconnect\n", err)
										break
									}
									_state = "ready"
									continue
								}
								statuRespBody, _ := ioutil.ReadAll(statuResp.Body)
								statuResp.Body.Close()
								if string(statuRespBody) != "success" {
									Error.Println("connect transit server error")
									defaulfMsg.Message.Status = "error"
									defaulfMsg.Message.Error = "p2p connect and transit server connect filed"
									err := ws.WriteJSON(defaulfMsg)
									if err != nil {
										Error.Printf("send connectResponse to registy err:%v, try to reconnect\n", err)
										break
									}
									_state = "ready"
									continue
								}
							}
							Info.Println("p2p mode connect successfully")
							defaulfMsg.Message.Fin = false
							err = ws.WriteJSON(defaulfMsg)
							if err != nil {
								Error.Printf("send connectResponse to registy err:%v, try to reconnect\n", err)
								break
							}
							_state = "operational"
							Info.Println("State: operational")
						}
					case "shutdown":
						_state = "ready"
						trans := ""
						if _, ok := m["trans"].(string); ok {
							trans = m["trans"].(string)
						}
						defaulfMsg := ConnectResp{
							Type:  "shutdown-response",
							Time:  fmt.Sprintf("%d", time.Now().Unix()),
							Trans: trans,
						}
						defaulfMsg.Entity.Role = "robot"
						defaulfMsg.Entity.ID = _config.Entity.ID
						defaulfMsg.Message.Fin = false
						req, _ := http.NewRequest("POST", "http://"+_config.AgentURI+"/shutdown", nil)
						_, err := http.DefaultClient.Do(req)
						if err != nil {
							Error.Printf("send shutdown request to agent err:%v\n", err)
							defaulfMsg.Message.Status = "error"
							defaulfMsg.Message.Error = fmt.Sprintf("send shutdown request to agent err:%v", err)
							err := ws.WriteJSON(defaulfMsg)
							if err != nil {
								Error.Printf("send shutdown-response to registy err:%v, try to reconnect\n", err)
								break
							}
							continue
						}
						defaulfMsg.Message.Status = "success"
						err = ws.WriteJSON(defaulfMsg)
						if err != nil {
							Error.Printf("send shutdown-response to registy err:%v, try to reconnect\n", err)
							break
						}
						Info.Println("shutdown success,State:ready")
					default:
						e := fmt.Sprintf("read json msg with an unexpected type:[%v]", m["type"])
						Error.Println(e)
						err := ws.WriteMessage(websocket.TextMessage, []byte("{\"error\":\""+e+"\"}"))
						if err != nil {
							Error.Printf("send Response to registy err:%v, try to reconnect\n", err)
							break
						}
					}

				}
			}
			cancel()
		}

	}

}
func getAddr() error {
	a, err := net.InterfaceAddrs()
	if err != nil {
		Error.Printf("get interfaceAddrs err:%v\n", err)
		return err
	}
	_addressInfo = AddressInfo{}
	for _, netAddr := range a {
		ipType := "udp4"
		if len(netAddr.String()) > 18 {
			ipType = "udp6"
		}
		pair := strings.Split(netAddr.String(), "/")
		_addressInfo.Addresses = append(_addressInfo.Addresses, Address{
			Type:      ipType,
			Address:   pair[0],
			Prefixlen: pair[1],
			Port:      _config.AgentBasePort,
		})
	}
	return nil
}
func GetRandomString(l int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyz"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < l; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}
