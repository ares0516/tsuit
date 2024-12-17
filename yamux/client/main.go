package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net"
	"sync"
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

func StartRing(conn net.Conn, msg string, interval int) {
	for {
		log.Printf("tik ==> %s", msg)
		conn.Write([]byte(msg))
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

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

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		StartBackServer(session)
	}()

	stream1, err := session.Open()
	if err != nil {
		log.Printf("Open stream1 error: %v", err)
	} else {
		wg.Add(1)
		go func() {
			defer wg.Done()
			StartRing(stream1, "Hello", 1)
		}()
	}

	stream2, err := session.Open()
	if err != nil {
		log.Printf("Open stream2 error: %v", err)
	} else {
		wg.Add(1)
		go func() {
			defer wg.Done()
			StartRing(stream2, "World", 5)
		}()
	}

	wg.Wait()
	return nil
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
