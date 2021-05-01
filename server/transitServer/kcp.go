package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/xtaci/kcp-go"
)

type myKcp struct {
	kcpUDPSession *kcp.UDPSession
	mtu           int
}

func NewMykcp(conn *kcp.UDPSession, mtu int) *myKcp {
	var m myKcp
	m.kcpUDPSession = conn
	m.mtu = mtu
	return &m
}
func (myKcp *myKcp) ReadMessage() ([]byte, error) {
	data := []byte{}
	for {
		b := make([]byte, myKcp.mtu)
		n, err := myKcp.kcpUDPSession.Read(b)
		if err != nil {
			log.Println(err)
			return []byte{}, nil
		}
		data = append(data, b[1:n]...)
		if b[0] == 0 {
			return data, nil
		}
	}

}
func (myKcp *myKcp) WriteMessage(msg []byte) error {
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
			log.Println(err)
			return err
		}
	}
	return nil
}
func (myKcp *myKcp) SetDeadline(t time.Time) error {
	err := myKcp.SetDeadline(t)
	return err
}
func (myKcp *myKcp) Close() error {
	err := myKcp.kcpUDPSession.Close()
	return err
}

func KcpListen(ctx context.Context, cancel context.CancelFunc, channel Channel, dispatcher *Dispatcher, side string) (int, error) {
	portMutex.Lock()
	defer portMutex.Unlock()
	for i := baseport; i < baseport+100; i++ {
		if _, ok := portUse[i]; !ok {
			laddr := net.JoinHostPort("0.0.0.0", fmt.Sprintf("%d", i))
			listener, err := kcp.ListenWithOptions(laddr, nil, 0, 0)
			if err != nil {
				return 0, err
			}
			log.Printf("kcp listen at %s\n", laddr)
			if side == "left" {
				go LeftKcpAccept(ctx, listener, channel, dispatcher)
			}
			if side == "right" {
				go RightKcpAccept(ctx, listener, channel, dispatcher)
			}
			portUse[i] = cancel
			return i, nil
		}
	}
	return 0, errors.New("no port can't use")
}
func LeftKcpAccept(ctx context.Context, listener *kcp.Listener, channel Channel, dispatcher *Dispatcher) {
	for {
		log.Println("KCP Accept start")
		conn_, err := listener.AcceptKCP()
		if err != nil {
			log.Println(err)
		}
		log.Println("KCP Accept succ")
		conn_.SetACKNoDelay(false)
		conn_.SetNoDelay(channel.Param.Nodelay[0], channel.Param.Nodelay[1], channel.Param.Nodelay[2], channel.Param.Nodelay[3])
		conn := NewMykcp(conn_, 1400)
		go leftReadHandleFunc(ctx, conn, dispatcher.leftInChan)
		go leftWriteHandleFunc(ctx, conn, dispatcher.leftOutChan[channel.Name])
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}
func RightKcpAccept(ctx context.Context, listener *kcp.Listener, channel Channel, dispatcher *Dispatcher) {
	for {
		log.Println("KCP Accept start")
		conn_, err := listener.AcceptKCP()
		if err != nil {
			log.Println(err)
		}
		log.Println("KCP Accept succ")
		conn_.SetACKNoDelay(false)
		conn_.SetNoDelay(channel.Param.Nodelay[0], channel.Param.Nodelay[1], channel.Param.Nodelay[2], channel.Param.Nodelay[3])
		conn := NewMykcp(conn_, 1400)
		go rightReadHandleFunc(ctx, conn, dispatcher.rightInChan)
		go rightWriteHandleFunc(ctx, conn, dispatcher.rightOutChan[channel.Name])
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}
