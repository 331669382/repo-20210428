package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	ConfigFile = "webConnCtl.config" //跟webConnCtl_robot相同
)

var _addressInfo AddressInfo
var _config Config
var _cert tls.Certificate

func main() {
	file, err := os.Open(ConfigFile)
	if err != nil {
		fmt.Printf("Open %s failed [Err:%v]", ConfigFile, err)
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
	_cert, err = tls.X509KeyPair([]byte(_config.Entity.Certificate), []byte(_config.Entity.Key))
	if err != nil {
		fmt.Printf("load cert err:%v", err)
		return
	}
	r := gin.Default()
	admin := r.Group("/admin")
	{
		admin.GET("/onlinelist", queryHandler)
		admin.POST("/connect", connectHandler)
		admin.GET("/status", statusHandlet)
	}
	err = r.Run(":" + _config.Adminport)
	if err != nil {
		fmt.Println(err)
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
