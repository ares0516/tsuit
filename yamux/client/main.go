package main

import (
	"crypto/tls"
	"flag"

	"github.com/ares0516/tsuit/common"
	"github.com/sirupsen/logrus"

	"github.com/hashicorp/yamux"
)

func Start(server string) error {

	config := &tls.Config{InsecureSkipVerify: true}
	conn, err := tls.Dial("tcp", server, config)
	if err != nil {
		return err
	}

	logrus.Info("remote conn peer address: %s", conn.RemoteAddr().String())

	err = common.PipeAuth(conn)
	if err != nil {
		logrus.Errorf("PipeAuth error: %v", err)
	} else {
		logrus.Info("PipeAuth success")
	}

	session, err := yamux.Client(conn, nil)
	if err != nil {
		return err
	}

	logrus.Info("remote session peer address: %s", session.RemoteAddr().String())

	logrus.Info("Waiting for connections....")

	socks5Server, err := common.NewSimpleSocksProxyServer()
	if err != nil {
		logrus.Errorf("NewSimpleSocksProxyServer error: %v", err)
	}

	for {
		stream, err := session.Accept()
		if err != nil {
			return err
		}
		logrus.Info("New back connection")
		go socks5Server.ServeConn(stream)
	}
}

func main() {
	server := flag.String("server", "192.168.31.142:1080", "The proxy server address)")
	flag.Parse()
	Start(*server)
}
