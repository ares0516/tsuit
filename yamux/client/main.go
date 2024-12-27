package main

import (
	"crypto/tls"
	"flag"
	"log"

	"github.com/ares0516/tsuit/common"

	"github.com/hashicorp/yamux"
)

func Start(server string) error {

	config := &tls.Config{InsecureSkipVerify: true}
	conn, err := tls.Dial("tcp", server, config)
	if err != nil {
		return err
	}

	log.Printf("remote conn peer address: %s", conn.RemoteAddr().String())

	session, err := yamux.Client(conn, nil)
	if err != nil {
		return err
	}

	log.Printf("remote session peer address: %s", session.RemoteAddr().String())

	log.Println("Waiting for connections....")

	socks5Server, err := common.NewSimpleSocksProxyServer()
	if err != nil {
		log.Printf("NewSimpleSocksProxyServer error: %v", err)
	}

	for {
		stream, err := session.Accept()
		if err != nil {
			return err
		}
		log.Println("New back connection")
		go socks5Server.ServeConn(stream)
	}
}

func main() {
	server := flag.String("server", "192.168.31.142:1080", "The proxy server address)")
	flag.Parse()
	Start(*server)
}
