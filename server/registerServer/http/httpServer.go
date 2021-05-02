package http

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"mcurobot.com/registerServer/common"
)

func RunHttpServer(errChan chan error) {
	cert, err := tls.LoadX509KeyPair(common.RegistryConfig.Servercertfile, common.RegistryConfig.Serverkeyfile)
	if err != nil {
		log.Fatalf("server: loadkeys: %s", err)
	}
	certpool := x509.NewCertPool()
	pem, err := ioutil.ReadFile(common.RegistryConfig.Clientcafile)
	if err != nil {
		log.Fatalf("Failed to read client certificate authority: %v", err)
	}
	if !certpool.AppendCertsFromPEM(pem) {
		log.Fatalf("Can't parse client certificate authority")
	}

	config := tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAnyClientCert,
		ClientCAs:    certpool,
	}

	r := gin.Default()
	r.POST("/register", registerHandler)
	r.POST("/query", queryHandler)
	r.POST("/connect", connectHandler)
	srv := &http.Server{Addr: common.RegistryConfig.Clientlisten, Handler: r, TLSConfig: &config}
	go func(errChan chan error) {
		err := srv.ListenAndServeTLS(common.RegistryConfig.Servercertfile, common.RegistryConfig.Serverkeyfile)
		errChan <- err
	}(errChan)
	return
}
