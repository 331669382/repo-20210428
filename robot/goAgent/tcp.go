package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

type myTcp struct {
	tcpConn net.Conn
	mtu     int
	w       sync.Mutex
	r       sync.Mutex
}

func NewMyTcp(conn net.Conn, mtu int) *myTcp {
	var m myTcp
	m.tcpConn = conn
	m.mtu = mtu
	m.w = sync.Mutex{}
	m.r = sync.Mutex{}
	return &m
}
func (myTcp *myTcp) ReadMessage() ([]byte, error) {
	data := []byte{}
	myTcp.r.Lock()
	defer myTcp.r.Unlock()
	for {
		b := make([]byte, myTcp.mtu)
		n, err := myTcp.tcpConn.Read(b)
		if err != nil {
			return []byte{}, err
		}
		data = append(data, b[1:n]...)
		if b[0] == 0 {
			return data, nil
		}
	}
}
func (myTcp *myTcp) WriteMessage(msg []byte) error {
	myTcp.w.Lock()
	defer myTcp.w.Unlock()
	for i := 0; i*(myTcp.mtu) < len(msg); i++ {
		var s []byte
		result := make([]byte, 1)
		if (i+1)*(myTcp.mtu) < len(msg) {
			result[0] = 1
			s = msg[i*(myTcp.mtu) : (i+1)*(myTcp.mtu)]
		} else {
			result[0] = 0
			s = msg[i*(myTcp.mtu):]
		}
		result = append(result, s...)
		_, err := myTcp.tcpConn.Write(result)
		if err != nil {
			return err
		}
	}
	return nil
}
func (myTcp *myTcp) SetDeadline(t time.Time) error {
	panic("SetDeadline")
}
func (myTcp *myTcp) Close() error {
	err := myTcp.tcpConn.Close()
	return err
}

func TcpListen(ctx context.Context, listen Listen, dispatcher *DispatcherStruct) error {
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
	listener, err := net.Listen("tcp", laddr)
	if err != nil {
		return err
	}
	Info.Printf("tcp start listen at %s\n", laddr)
	go TcpAccept(ctx, listener, curChannel, dispatcher)
	return nil
}
func TcpAccept(ctx context.Context, listener net.Listener, curChannel Channel, dispatcher *DispatcherStruct) {
	for {
		Info.Println("tcp Accept start")
		conn_, err := listener.Accept()
		if err != nil {
			Warning.Printf("tcp listener.Accepttcp error:%v\n", err)
			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}
		conn := NewMyTcp(conn_, 1400)
		Info.Println("tcp Accept succ")
		if _, ok := connMapL[curChannel.Name]; ok {
			connMapL[curChannel.Name] = append(connMapL[curChannel.Name].([]*myTcp), conn)
		} else {
			connMapL[curChannel.Name] = []*myTcp{conn}
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

func TcpDial(ctx context.Context, dial Dial, dispatcher *DispatcherStruct, isFish bool) error {
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
	Info.Printf("Tcp try to dial:%s\n", raddr)
	conn_, err := net.Dial("tcp", raddr)
	if err != nil {
		return err
	}
	conn := NewMyTcp(conn_, 1400)
	if _, ok := connMapD[curChannel.Name]; ok {
		connMapDlock.Lock()
		connMapD[curChannel.Name] = append(connMapD[curChannel.Name].([]*myTcp), conn)
		if !isFish {
			isFish = true
			dialFinish <- struct{}{}
		}
		connMapDlock.Unlock()
	} else {
		connMapDlock.Lock()
		connMapD[curChannel.Name] = []*myTcp{conn}
		connMapDlock.Unlock()
	}
	go remoteReadHandleFunc(ctx, conn, dispatcher.remoteInChan[curChannel.Name], curChannel.Name)
	go remoteWriteHandleFunc(ctx, conn, dispatcher.remoteOutChan[curChannel.Name], curChannel.Name)

	return nil
}
