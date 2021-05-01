package main

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func WsConnect(ctx context.Context, URI string, dispatcher *DispatcherStruct) error {
	Info.Printf("Websocket try to connect URI:%s\n", URI)
	ws, _, err := websocket.DefaultDialer.Dial(URI, nil)
	//ws, err := websocket.Dial(URI, "", "http://localhost")
	if err != nil {
		return err
	}
	go localReadHandleFunc(ctx, ws, dispatcher.localInChan)
	go localWriteHandleFunc(ctx, ws, dispatcher.localOutChan)
	return nil
}
func WsRunServer(ctx context.Context, URI string, dispatcher *DispatcherStruct) error {
	uri, err := url.Parse(URI)
	if err != nil {
		return err
	}
	if uri.Path == "" {
		uri.Path = "/"
	}

	childCtx, childCancel := context.WithCancel(ctx)
	mux := http.NewServeMux()
	mux.HandleFunc(uri.Path, func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			Warning.Printf("Websocket upgrade error:%v\n", err)
			return
		}
		go checkStat(childCtx, childCancel)
		go localReadHandleFunc(childCtx, ws, dispatcher.localInChan)
		go localWriteHandleFunc(childCtx, ws, dispatcher.localOutChan)

		select {
		case <-ctx.Done():
			return
		}

	})
	srv := &http.Server{Addr: uri.Host, Handler: mux}
	go func(ctx context.Context, srv *http.Server) {
		go srv.ListenAndServe()
		Info.Printf("Websocket start listen at %s\n", uri.Host)

		select {
		case <-ctx.Done():
			srv.Close()
			Info.Println("Websocket server closed")
			return
			//srv.Shutdown(nil)
		}

	}(ctx, srv)
	return nil
}
func checkStat(ctx context.Context, cancel context.CancelFunc) {

}
