package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/xtaci/kcp-go"
)

type myKcp struct {
	kcpUDPSession *kcp.UDPSession
	mtu           int
	w             sync.Mutex
	r             sync.Mutex
}

func NewMykcp(conn *kcp.UDPSession, mtu int) *myKcp {
	var m myKcp
	m.kcpUDPSession = conn
	m.mtu = mtu
	m.w = sync.Mutex{}
	m.r = sync.Mutex{}
	return &m
}
func (myKcp *myKcp) ReadMessage() ([]byte, error) {
	data := []byte{}
	myKcp.r.Lock()
	defer myKcp.r.Unlock()
	for {
		b := make([]byte, myKcp.mtu)
		n, err := myKcp.kcpUDPSession.Read(b)
		if err != nil {
			return []byte{}, err
		}
		data = append(data, b[1:n]...)
		if b[0] == 0 {
			return data, nil
		}
	}

}
func (myKcp *myKcp) WriteMessage(msg []byte) error {
	myKcp.w.Lock()
	defer myKcp.w.Unlock()
	for i := 0; i*(myKcp.mtu-33) < len(msg); i++ {
		var s []byte
		result := make([]byte, 1)
		if (i+1)*(myKcp.mtu-33) < len(msg) {
			result[0] = 1
			s = msg[i*(myKcp.mtu-33) : (i+1)*(myKcp.mtu-33)]
		} else {
			result[0] = 0
			s = msg[i*(myKcp.mtu-33):]
		}
		result = append(result, s...)
		_, err := myKcp.kcpUDPSession.Write(result)
		if err != nil {
			return err
		}
	}
	return nil
}
func (myKcp *myKcp) SetDeadline(t time.Time) error {
	err := myKcp.kcpUDPSession.SetDeadline(t)
	return err
}
func (myKcp *myKcp) Close() error {
	err := myKcp.kcpUDPSession.Close()
	return err
}

func KcpListen(ctx context.Context, listen Listen, dispatcher *DispatcherStruct) error {
	var curChannel Channel
	for _, channel := range CFG.Channels {
		if channel.Name == listen.Name && channel.Proto == listen.Proto {
			curChannel = channel
		}
	}
	if curChannel.Name == "" {
		return errors.New(fmt.Sprintf("no %s channel called %s,plz send config first", listen.Proto, listen.Name))
	}
	laddr := net.JoinHostPort(listen.Host, fmt.Sprintf("%d", listen.Port))
	listener, err := kcp.ListenWithOptions(laddr, nil, 0, 0)
	if err != nil {
		return err
	}
	Info.Printf("kcp start listen at %s\n", laddr)
	go KcpAccept(ctx, listener, curChannel, dispatcher)
	return nil
}
func KcpAccept(ctx context.Context, listener *kcp.Listener, curChannel Channel, dispatcher *DispatcherStruct) {
	for {
		Info.Println("KCP Accept start")
		conn_, err := listener.AcceptKCP()
		if err != nil {
			Warning.Printf("KCP listener.AcceptKCP error:%v\n", err)
			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}
		Info.Println("KCP Accept succ")
		conn_.SetACKNoDelay(_config.KcpNodelay)
		conn_.SetNoDelay(curChannel.Param.Nodelay[0], curChannel.Param.Nodelay[1], curChannel.Param.Nodelay[2], curChannel.Param.Nodelay[3])
		conn_.SetWindowSize(curChannel.Param.Window[0], curChannel.Param.Window[1])
		conn := NewMykcp(conn_, 1400)
		conn.WriteMessage([]byte("ping"))
		if _, ok := connMapL[curChannel.Name]; ok {
			connMapL[curChannel.Name] = append(connMapL[curChannel.Name].([]*myKcp), conn)
		} else {
			connMapL[curChannel.Name] = []*myKcp{conn}
			if len(connMapL) == len(CFG.Channels) {
				listenFinish <- struct{}{}
			}
		}
		go remoteReadHandleFunc(ctx, conn, dispatcher.remoteInChan[curChannel.Name], curChannel.Name)
		go remoteWriteHandleFunc(ctx, conn, dispatcher.remoteOutChan[curChannel.Name], curChannel.Name)
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}
func KcpDial(ctx context.Context, dial Dial, dispatcher *DispatcherStruct, isFish bool) error {
	var curChannel Channel
	for _, channel := range CFG.Channels {
		if channel.Name == dial.Name && channel.Proto == dial.Proto {
			curChannel = channel
			break
		}
	}
	if curChannel.Name == "" {
		return errors.New(fmt.Sprintf("no %s channel called %s,plz send config first", dial.Proto, dial.Name))
	}
	raddr := net.JoinHostPort(dial.Host, fmt.Sprintf("%d", dial.Port))
	Info.Printf("Kcp try to dial:%s\n", raddr)
	conn_, err := kcp.DialWithOptions(raddr, nil, 0, 0)
	if err != nil {
		return err
	}
	conn_.SetACKNoDelay(_config.KcpNodelay)
	conn_.SetNoDelay(curChannel.Param.Nodelay[0], curChannel.Param.Nodelay[1], curChannel.Param.Nodelay[2], curChannel.Param.Nodelay[3])
	conn_.SetWindowSize(curChannel.Param.Window[0], curChannel.Param.Window[1])
	conn := NewMykcp(conn_, 1400)
	err = conn.WriteMessage([]byte("ping"))
	if err != nil {
		return err
	}
	go func() {
		_, err := conn.ReadMessage()
		if err == nil {
			Info.Printf("Kcp dial succ:%s\n", raddr)
			if _, ok := connMapD[curChannel.Name]; ok {
				connMapDlock.Lock()
				connMapD[curChannel.Name] = append(connMapD[curChannel.Name].([]*myKcp), conn)
				if !isFish {
					isFish = true
					dialFinish <- struct{}{}
				}
				connMapDlock.Unlock()
			} else {
				connMapDlock.Lock()
				connMapD[curChannel.Name] = []*myKcp{conn}
				connMapDlock.Unlock()
			}
			go remoteReadHandleFunc(ctx, conn, dispatcher.remoteInChan[curChannel.Name], curChannel.Name)
			go remoteWriteHandleFunc(ctx, conn, dispatcher.remoteOutChan[curChannel.Name], curChannel.Name)
		}
	}()

	return nil
}
