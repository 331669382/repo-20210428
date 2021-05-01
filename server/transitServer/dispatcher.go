package main

import (
	"context"
	"encoding/json"
	"log"
)

func newDispatcher(Channels []Channel) *Dispatcher {
	var dispatcher Dispatcher
	dispatcher.leftInChan = make(chan Message)
	dispatcher.leftOutChan = make(map[string]chan Message)
	dispatcher.rightInChan = make(chan Message)
	dispatcher.rightOutChan = make(map[string]chan Message)
	for _, channel := range Channels {
		dispatcher.leftOutChan[channel.Name] = make(chan Message)
		dispatcher.rightOutChan[channel.Name] = make(chan Message)
	}
	return &dispatcher
}
func dispatchWorker(ctx context.Context, str *Dispatcher, Channels []Channel) {
	go func(ctx context.Context, str *Dispatcher) {
		for {
			select {
			case msg := <-str.leftInChan:
				if string(msg) == "ping" || string(msg) == "closed" {
					for _, chann := range str.rightOutChan {
						chann <- msg
					}
					continue
				}
				msgtype, topic := getMsgType(msg)
				chNames := decideChan(msgtype, topic, Channels)
				if msgtype != "echo" {
					log.Println(chNames, topic)
				}
				for _, chName := range chNames {
					select {
					case str.rightOutChan[chName] <- msg:
						//case <-time.After(time.Millisecond * 100):

					}

				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx, str)

	go func(ctx context.Context, str *Dispatcher) {
		for {
			select {
			case msg := <-str.rightInChan:
				if string(msg) == "ping" || string(msg) == "closed" {
					for _, chann := range str.leftOutChan {
						chann <- msg
					}
					continue
				}
				msgtype, topic := getMsgType(msg)
				chNames := decideChan(msgtype, topic, Channels)
				log.Println(chNames, topic)
				for _, chName := range chNames {
					select {
					case str.leftOutChan[chName] <- msg:
						//case <-time.After(time.Millisecond * 100):

					}

				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx, str)

}

func getMsgType(msg Message) (msgtype string, topic string) {
	m := make(map[string]interface{})
	if err := json.Unmarshal(msg, &m); err != nil {
		log.Println(err)
		return "", ""
	}
	if _, ok := m["op"]; ok {
		if m["op"] == "echo" || m["op"] == "echoresponse" {
			if _, ok_ := m["channel"]; ok_ {
				return "echo", m["channel"].(string)
			}
			return "", ""
		}
		if _, ok_ := m["topic"]; ok_ {
			return "op", m["topic"].(string)
		}

		return "op", ""
	}

	return "frag", ""
}
func decideChan(msgtype string, topic string, Channels []Channel) (chNames []string) {
	switch msgtype {
	case "op":
		for _, channel := range Channels {
			if channel.OpChannel {
				for _, sub := range channel.Subscription {
					if sub == "*" || sub == topic {
						chNames = append(chNames, channel.Name)
						return
					}
				}
			}
		}
	case "frag":
		for _, channel := range Channels {
			if channel.FragChannel {
				for _, sub := range channel.Subscription {
					if sub == "*" || sub == topic {
						chNames = append(chNames, channel.Name)
						return
					}
				}
			}
		}
	case "echo":
		for _, channel := range Channels {
			if channel.Name == topic {
				chNames = append(chNames, channel.Name)
				return
			}
		}
	}
	return
}
func leftWriteHandleFunc(ctx context.Context, conn MessageConn, leftOutChan <-chan Message) {
	for {
		select {
		case msg := <-leftOutChan:
			err := conn.WriteMessage(msg)
			if err != nil {
				log.Println(err)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func leftReadHandleFunc(ctx context.Context, conn MessageConn, leftInChan chan<- Message) {
	for {
		msg, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		select {
		case leftInChan <- msg:
		//case <-time.After(time.Millisecond * 100):
		//fmt.Println("timeout")
		case <-ctx.Done():
			break
		}
	}

}

func rightWriteHandleFunc(ctx context.Context, conn MessageConn, rightOutChan <-chan Message) {
	for {
		select {
		case msg := <-rightOutChan:
			err := conn.WriteMessage(msg)
			if err != nil {
				log.Println(err)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func rightReadHandleFunc(ctx context.Context, conn MessageConn, rightInChan chan<- Message) {
	for {
		msg, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		select {
		case rightInChan <- msg:
		//case <-time.After(time.Millisecond * 100):
		//fmt.Println("timeout")
		case <-ctx.Done():
			break
		}
	}

}
