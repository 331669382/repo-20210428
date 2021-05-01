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
					case <-time.After(time.Second * 290):
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
					case "connect":
						connectRequest := ConnectReq{}
						err := json.Unmarshal(msg, &connectRequest)
						if err != nil {
							Error.Printf("read json msg with a wrong connectRequest:%v, msg:%s\n", err, string(msg))
							continue
						}
						defaulfMsg := ConnectResp{
							Version: connectRequest.Version,
							Type:    "connect-response",
							Time:    fmt.Sprintf("%d", time.Now().Unix()),
							Trans:   connectRequest.Trans,
						}
						defaulfMsg.Entity.Role = "robot"
						defaulfMsg.Entity.ID = _config.Entity.ID
						defaulfMsg.Message.Fin = false
						for i, Authorizedclient := range _config.Authorizedclients {
							if connectRequest.Entity.ID == Authorizedclient {
								break
							}
							if i == len(_config.Authorizedclients) {
								Error.Printf("unAuthorizedclient\n")
								defaulfMsg.Message.Status = "error"
								defaulfMsg.Message.Error = "unauthorized client"
								err := ws.WriteJSON(defaulfMsg)
								if err != nil {
									Error.Printf("send connectResponse to registy err:%v, try to reconnect\n", err)
									break
								}
								continue
							}
						}

						switch connectRequest.Message.Status {
						case "new":
							_state = "connecting"
							Info.Println("get a connectRequest,State: connecting")
							if len(connectRequest.Transit.Port) != len(_defaultChannel) {
								Error.Printf("len(connectRequest.Transit.Port)!=len(defaultChannel)\n")
								defaulfMsg.Message.Status = "error"
								defaulfMsg.Message.Error = "len(connectRequest.Transit.Port)!=len(defaultChannel)"
								err := ws.WriteJSON(defaulfMsg)
								if err != nil {
									Error.Printf("send connectResponse to registy err:%v, try to reconnect\n", err)
									break
								}
								continue
							}
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
								continue
							}
							confRespBody, _ := ioutil.ReadAll(confResp.Body)
							confRespJSON := NormalResp{}
							err = json.Unmarshal(confRespBody, &confRespJSON)
							if err != nil {
								Error.Printf("recv connectResp from agent err:%v,body:%s\n", err, confRespBody)
								defaulfMsg.Message.Status = "error"
								defaulfMsg.Message.Error = fmt.Sprintf("recv connectResp from agent err:%v,body:%s", err, confRespBody)
								err := ws.WriteJSON(defaulfMsg)
								if err != nil {
									Error.Printf("send connectResponse to registy err:%v, try to reconnect\n", err)
									break
								}
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
								continue
							}
							conn := AgentConnect{
								Role:           "server",
								Cleanupoldtask: clean,
							}
							if _config.Entity.Role == "client" {
								conn.Role = "client"
							}
							for i, c := range _defaultChannel {
								conn.Dial = append(conn.Dial, DialOrListen{
									Name:  c.Name,
									Proto: c.Proto,
									Host:  connectRequest.Transit.Host,
									Port:  connectRequest.Transit.Port[i],
								})
								conn.Listen = append(conn.Listen, DialOrListen{
									Name:  c.Name,
									Proto: c.Proto,
									Host:  "0.0.0.0",
									Port:  _config.AgentBasePort + i,
								})
							}
							connectReqBody, _ := json.Marshal(conn)
							connectReq, _ := http.NewRequest("POST", "http://"+_config.AgentURI+"/conn/connect", bytes.NewBuffer(connectReqBody))
							connResp, err := http.DefaultClient.Do(connectReq)
							if err != nil {
								Error.Printf("send connect request to agent err:%v\n", err)
								defaulfMsg.Message.Status = "error"
								defaulfMsg.Message.Error = fmt.Sprintf("send connect request to agent err:%v", err)
								err := ws.WriteJSON(defaulfMsg)
								if err != nil {
									Error.Printf("send connectResponse to registy err:%v, try to reconnect\n", err)
									break
								}
								continue
							}
							connRespBody, _ := ioutil.ReadAll(connResp.Body)
							connRespJSON := NormalResp{}
							json.Unmarshal(connRespBody, &connRespJSON)
							if connRespJSON.Code != succCode {
								defaulfMsg.Message.Status = "error"
								Error.Printf("agent connect error %s\n", connRespJSON.Msg)
								defaulfMsg.Message.Error = fmt.Sprintf("agent connect error %s", connRespJSON.Msg)
								err := ws.WriteJSON(defaulfMsg)
								if err != nil {
									Error.Printf("send connectResponse to registy err:%v, try to reconnect\n", err)
									break
								}
							}
							defaulfMsg.Message.Status = "pending"
							err = ws.WriteJSON(defaulfMsg)
							if err != nil {
								Error.Printf("send connectResponse to registy err:%v, try to reconnect\n", err)
								break
							}
							_state = "operational"
							Info.Println("State: operational")
						case "preauth":
						}
					case "shutdown":
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
						_state = "ready"
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
