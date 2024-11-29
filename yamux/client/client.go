package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net"
	"time"

	"github.com/hashicorp/yamux"
)

func StartBackServer(session *yamux.Session) error {
	for {
		stream, err := session.Accept()
		if err != nil {
			return err
		}
		log.Println("New connection")
		go handleStream(stream)
	}
}

func Start(server string) error {
	config := &tls.Config{InsecureSkipVerify: true}
	conn, err := tls.Dial("tcp", server, config)
	if err != nil {
		return err
	}

	session, err := yamux.Client(conn, nil)
	if err != nil {
		return err
	}

	log.Println("Waiting for connections....")

	go StartBackServer(session)

	for {
		stream, err := session.Open()
		if err != nil {
			return err
		}
		stream.Write([]byte("Hello, world!"))
		time.Sleep(3 * time.Second)
	}
}

func handleStream(stream net.Conn) {
	defer stream.Close()
	buf := make([]byte, 1024)
	for {
		n, err := stream.Read(buf)
		if err != nil {
			return
		}
		log.Println("Received:", string(buf[:n]))
	}
}

func main() {
	server := flag.String("server", "192.168.31.142:1080", "The relay server (the connect-back address)")
	flag.Parse()

	Start(*server)
}
