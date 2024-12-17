package main

import (
	"crypto/tls"
	"errors"
	"log"
	"net"

	"github.com/hashicorp/yamux"
)

const (
	// 监听地址
	ListenAddr = "0.0.0.0"
	// 监听端口
	ListenPort = "1080"
	// 证书文件
	CertFile = "../cert/test.crt"
	// 私钥文件
	KeyFile = "../cert/test.key"
)

const (
	ENABLE_TLS  = true
	DISABLE_TLS = false
)

func Start(enableTLS bool) {
	var listener net.Listener
	var err error

	if enableTLS {
		// 加载证书和私钥
		cert, err := tls.LoadX509KeyPair(CertFile, KeyFile)
		if err != nil {
			log.Fatalf("无法加载证书和私钥: %v", err)
		}

		// 创建 TLS 配置
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		// 创建 TLS 监听器
		listener, err = tls.Listen("tcp", ListenAddr+":"+ListenPort, tlsConfig)
		if err != nil {
			log.Fatalf("无法监听端口: %v", err)
		}
		log.Println("YAMUX TLS 服务器正在监听 " + ListenAddr + ":" + ListenPort)
	} else {
		// 创建普通 TCP 监听器
		listener, err = net.Listen("tcp", ListenAddr+":"+ListenPort)
		if err != nil {
			log.Fatalf("无法监听端口: %v", err)
		}
		log.Println("YAMUX 服务器正在监听 " + ListenAddr + ":" + ListenPort)
	}
	defer listener.Close()

	log.Println("等待客户端连接...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("接受连接失败: %v", err)
			continue
		}

		log.Printf("接受来自 %s 的连接", conn.RemoteAddr())

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) error {
	session, err := yamux.Server(conn, nil)
	if err != nil {
		log.Printf("创建 yamux 会话失败: %v", err)
		return err
	}
	defer session.Close()

	for {
		if session.IsClosed() {
			log.Println("会话已关闭")
			return nil
		}
		stream, err := session.Accept()

		if err != nil {
			if errors.Is(err, yamux.ErrConnectionReset) {
				log.Println("连接已重置")
				return nil
			}
			log.Printf("接受 yamux 流失败: %v", err)
			continue
		}
		log.Printf("接受来自 %s 的流", stream.RemoteAddr())
		go handleStream(stream)
	}
}

func handleStream(stream net.Conn) {
	defer stream.Close()

	buf := make([]byte, 1024)
	for {
		n, err := stream.Read(buf)
		if err != nil {
			log.Printf("读取数据失败: %v", err)
			return
		}
		log.Println("接收到数据:", string(buf[:n]))
	}
}

func main() {
	Start(ENABLE_TLS)
}
