package websocket

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"mcurobot.com/registerServer/common"
	"mcurobot.com/registerServer/log"
)

var OnlineRobotWs map[string]*websocket.Conn = make(map[string]*websocket.Conn)
var RobotInfo map[string]common.AddressInfo = make(map[string]common.AddressInfo)

func RunWsServer(conf common.Config, addr string, errChan chan error) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var upgrader = websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			//Subprotocols: []string{r.Header.Get("Sec-Websocket-Protocol")},
		}
		if r.TLS == nil || r.TLS.PeerCertificates[0] == nil || r.TLS.PeerCertificates[0].Subject.CommonName == "" {
			return
		}
		robotName := r.TLS.PeerCertificates[0].Subject.CommonName
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Warning.Printf("Ws Upgrade err:%v\n", err)
			return
		}
		log.Info.Printf("robot [%s] connect,ready to receive register request\n", robotName)
		j := common.RegisterReq{}
		resp := common.RegisterResp{
			Type: "register-response",
		}
		resp.Entity.Role = "registry"
		err = ws.ReadJSON(&j)
		if err != nil {
			e := fmt.Sprintf("read robot [%s] register json message error :%s", robotName, err)
			log.Error.Println(e)
			resp.Message.Status = "error"
			resp.Message.Error = e
			err_ := ws.WriteJSON(resp)
			if err_ == nil {
				ws.Close()
				e_ := fmt.Sprintf("send msg to robot [%s] error:%s,connection closed", robotName, err_)
				log.Error.Println(e_)
			}
			return
		}
		RobotInfo[robotName] = j.Message
		OnlineRobotWs[robotName] = ws
		log.Info.Printf("robot client [%s] register success\n", robotName)
		resp.Trans = j.Trans
		resp.Version = j.Version
		resp.Message.Status = "success"
		err = ws.WriteJSON(resp)
		if err != nil {
			ws.Close()
			e_ := fmt.Sprintf("send msg to robot [%s] error:%s,connection closed", robotName, err)
			log.Error.Println(e_)
			return
		}
		for {
			select {
			case <-time.After(time.Second):
				if recordedWs, ok := OnlineRobotWs[robotName]; ok {
					if recordedWs != ws {
						log.Warning.Printf("Maybe two robot client use same robotName %s\n", robotName)
						ws.WriteMessage(websocket.TextMessage, []byte("another robot  use your robotName to register,you have been offline"))
						return
					}
					if err := ws.WriteMessage(websocket.PingMessage, []byte("")); err != nil {
						delete(OnlineRobotWs, robotName)
						log.Info.Printf("robot  [%s] disconnected", robotName)
						return
					}
				} else {
					return
				}
			}
		}
	})
	cert, err := tls.LoadX509KeyPair(conf.Servercertfile, conf.Serverkeyfile)
	if err != nil {
		return errors.New(fmt.Sprintf("server: loadkeys: %v", err))
	}
	certpool := x509.NewCertPool()
	pem, err := ioutil.ReadFile(conf.Clientcafile)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to read client certificate authority: %v", err))
	}
	if !certpool.AppendCertsFromPEM(pem) {
		return errors.New("Can't parse client certificate authority")
	}
	config := tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certpool,
	}
	srv := &http.Server{Addr: addr, Handler: mux, TLSConfig: &config}
	go func(srv *http.Server, errChan chan error) {
		log.Info.Printf("Websocket server start listen at %s\n", addr)
		err := srv.ListenAndServeTLS(conf.Servercertfile, conf.Serverkeyfile)
		errChan <- err
	}(srv, errChan)
	return nil
}
