package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net"
	"sync"
	"syscall"

	"github.com/ares0516/tsuit/common"
	"github.com/hashicorp/yamux"
	"github.com/sirupsen/logrus"
)

const (
	// 监听地址
	ListenAddr = "0.0.0.0"
	// 监听�?�?
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

type Server struct {
	sync.Mutex
	LocalAddress string // 本地地址
	EntryAddress string // 入口地址
	AddressMap   map[string]*yamux.Session
}

var _manager = common.NewManager()

func NewServer(localAddress, entryAddress string) *Server {
	return &Server{
		LocalAddress: localAddress,
		EntryAddress: entryAddress,
	}
}

func (s *Server) startEntryServer() {
	cert, err := tls.LoadX509KeyPair(CertFile, KeyFile)
	if err != nil {
		logrus.Errorf("加载证书失败: %v", err)
		return
	}

	config := &tls.Config{Certificates: []tls.Certificate{cert}}

	listener, err := tls.Listen("tcp", s.EntryAddress, config)
	if err != nil {
		logrus.Errorf("监听入口地址失败: %v", err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			logrus.Errorf("接受连接失败: %v", err)
			continue
		}
		go s.handleEntryConnection(conn)
	}
}

func (s *Server) startLocalServer() {
	conn, err := net.Listen("tcp", s.LocalAddress)
	if err != nil {
		logrus.Errorf("监听本地地址失败: %v", err)
		return
	}
	defer conn.Close()

	for {
		client, err := conn.Accept()
		if err != nil {
			logrus.Errorf("接受本地连接失败: %v", err)
			continue
		}
		go s.handleLocalConnection(client)
	}

}

func (s *Server) handleLocalConnection(conn net.Conn) {
	defer conn.Close()
	logrus.WithFields(logrus.Fields{"local address": conn.LocalAddr()}).Info("New local connection.\n")
	destAddr, port, err := GetOriginalDst(conn)
	if err != nil {
		logrus.Errorf("Failed to get original destination: %v", err)
		return
	}
	logrus.WithFields(logrus.Fields{"dest address": destAddr}).Info("New local connection.\n")

	session := _manager.Get(destAddr)

	if session != nil {
		logrus.Warningf("No session found for IP address %s, closing connection from %s", destAddr, conn.RemoteAddr())
		conn.Close()
		return
	}

	stream, err := session.Open()
	if err != nil {
		logrus.Errorf("Could not open session : %s\n", err)
		return
	}

	//在stream上做socks5认证
	common.Auth(stream)

	//建立socks连接
	common.Requisition(stream, "127.0.0.1", port, common.Connect)

	go func() {
		io.Copy(conn, stream)
		conn.Close()
	}()
	io.Copy(stream, conn)
}

// 处理客户端传入链接
func (s *Server) handleEntryConnection(conn net.Conn) (*yamux.Session, error) {
	logrus.WithFields(logrus.Fields{"remoteaddr": conn.RemoteAddr().String()}).Info("New relay connection.\n")
	session, err := yamux.Server(conn, nil)
	if err != nil {
		return nil, err
	}
	ping, err := session.Ping()
	if err != nil {
		return nil, err
	}
	logrus.Printf("Session ping : %v\n", ping)

	vip := allocAddr()
	_manager.Add(vip, session)
	_manager.Dump()

	return session, nil
}

func allocAddr() string {
	for i := 1; i < 255; i++ {
		addr := fmt.Sprintf("10.0.0.%d", i)
		if !_manager.IsExist(addr) {
			return addr
		}
	}
	return ""
}

func main() {
	localAddress := flag.String("local", "0.0.0.0:5555", "The local address")
	entryAddress := flag.String("entry", "0.0.0.0:1080", "The entry address")
	flag.Parse()
	server := NewServer(*localAddress, *entryAddress)
	go server.startEntryServer()
	server.startLocalServer()
}

func GetOriginalDst(conn net.Conn) (string, uint16, error) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return "", 0, fmt.Errorf("not a TCP connection")
	}

	file, err := tcpConn.File()
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	fd := int(file.Fd())
	addr, err := syscall.GetsockoptIPv6Mreq(fd, syscall.IPPROTO_IP, 80)
	if err != nil {
		return "", 0, err
	}

	ip := net.IPv4(addr.Multiaddr[4], addr.Multiaddr[5], addr.Multiaddr[6], addr.Multiaddr[7])
	port := uint16(addr.Multiaddr[2])<<8 + uint16(addr.Multiaddr[3])

	return ip.String(), port, nil
}
