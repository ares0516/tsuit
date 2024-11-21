package main

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
)

type AuthRequest struct {
	Version byte
	Token   string
	ResID   string
}

type ConnRequest struct {
	Cmd       uint8
	Resverd   uint8
	Addr      *Addr
	ExtraLen  uint16
	ExtraData ExtData
}

type ExtData struct {
	Version int `json:"version"`
	Data    struct {
		ClientIp    string `json:"client_ip"`
		ProcessName string `json:"process_name"`
		OS          string `json:"os"`
	} `json:"data"`
}

type Addr struct {
	Type uint8
	Host string
	Port uint16
}

const (
	// SOCKS5 协议版本
	Version = 0x05

	Success     = 0x00
	Unreachable = 0x03

	//
	MethodToken        = 0x80
	MethodNoAcceptable = 0xFF

	// 监听地址
	ListenAddr = "0.0.0.0"
	// 监听端口
	ListenPort = "1080"
	// 证书文件
	CertFile = "../resource/test.crt"
	// 私钥文件
	KeyFile = "../resource/test.key"
)

var (
	ErrBadVersion = errors.New("bad version")
	ErrBadMethod  = errors.New("bad method")
)

func socks_start(useTLS bool) {
	var listener net.Listener
	var err error

	if useTLS {
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
		log.Println("SOCKS5 TLS 服务器正在监听 " + ListenAddr + ":" + ListenPort)
	} else {
		// 创建普通 TCP 监听器
		listener, err = net.Listen("tcp", ListenAddr+":"+ListenPort)
		if err != nil {
			log.Fatalf("无法监听端口: %v", err)
		}
		log.Println("SOCKS5 服务器正在监听 " + ListenAddr + ":" + ListenPort)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("接受连接失败: %v", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) error {
	defer conn.Close()

	// handshake
	if err := ServerHandshake(conn); err != nil {
		log.Printf("协商失败: %v", err)
		return err
	}
	log.Printf("协商成功")

	// authentication
	_, err := Authentication(conn)
	if err != nil {
		log.Printf("认证失败: %v", err)
		return err
	}

	conn.Write([]byte{MethodToken, 0x00}) // 认证成功

	// connection
	connReq, err := Connection(conn)
	if err != nil {
		log.Printf("连接失败: %v", err)
		return err
	}

	switch connReq.Cmd {
	case 0x05: // CMD_GATEWAY_STATE

		handlerCmdGatewaySate(conn)
	case 0x01: // CMD_CONNECT
		handlerCmdConnect(conn, connReq)
	}

	// // 代理连接
	// pconn, err := net.Dial("tcp", connReq.Addr.Host+":"+string(connReq.Addr.Port))
	// if err != nil {
	// 	log.Printf("连接目标服务器失败: %v", err)
	// 	return err
	// }

	conn.Write([]byte{Version, 0x00, 0x00, connReq.Addr.Type, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	return nil
}

// +----+----------+----------+
// |VER | NMETHODS | METHODS  |
// +----+----------+----------+
// | 1  |    1     | 1 to 255 |
// +----+----------+----------+
// ServerHandshake 处理客户端连接

func ServerHandshake(conn net.Conn) error {
	// 读取协商数据
	buf := make([]byte, 258)
	if _, err := io.ReadAtLeast(conn, buf, 3); err != nil {
		return err
	}

	// 检查协议版本
	if buf[0] != Version {
		conn.Write([]byte{Version, MethodNoAcceptable})
		log.Printf("bad version: %d", buf[0])
		return ErrBadVersion
	}

	// 检查认证方法数量
	nMethods := int(buf[1])
	// if nMethods == 0 || nMethods > 255 {
	if nMethods != 1 { // 只支持 1 种认证方法
		conn.Write([]byte{Version, MethodNoAcceptable})
		return ErrBadMethod
	}

	// 检查是否为 Token 认证
	methods := buf[2 : 2+nMethods]
	supportsToken := false
	for _, method := range methods {
		if method == MethodToken {
			supportsToken = true
			break
		}
	}

	if !supportsToken {
		conn.Write([]byte{Version, MethodNoAcceptable})
		return ErrBadMethod
	}

	log.Printf("version: %x, nMethods: %x, methods: %x", buf[0], nMethods, methods)

	// 选择 Token 认证
	conn.Write([]byte{Version, MethodToken})
	return nil
}

// +----+-----------+----------+-----------+----------+
// |VER | TOKEN_LEN |  TOKEN   | RESID_LEN |  RESID   |
// +----+-----------+----------+-----------+----------+
// |byte|  byte     | string   |  byte     | string   |
// +----+-----------+----------+-----------+----------+
func Authentication(conn net.Conn) (*AuthRequest, error) {
	offset := 0
	req := &AuthRequest{}
	// 读取协商数据
	buf := make([]byte, 512)
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return nil, err
	}

	if buf[offset] != MethodToken {
		return nil, ErrBadMethod
	}
	offset += 1

	tokenLen := int(buf[offset])
	offset += 1

	// addition 1 byte for user id length
	if _, err := io.ReadFull(conn, buf[offset:offset+tokenLen+1]); err != nil {
		return nil, err
	}
	token := string(buf[offset : offset+tokenLen])
	offset += tokenLen

	ridLen := int(buf[offset])
	offset += 1

	if _, err := io.ReadFull(conn, buf[offset:offset+ridLen]); err != nil {
		return nil, err
	}
	rid := string(buf[offset : offset+ridLen])

	req.Version = Version
	req.Token = token
	req.ResID = rid

	log.Printf("Token: %s, ResID: %s", req.Token, req.ResID)

	return req, nil
}

// +----+-----+-------+------+----------+----------+----------+----------+
// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT | EXT.LEN  | EXT.DATA |
// +----+-----+-------+------+----------+----------+----------+----------+
// | 1  |  1  | X'00' |  1   | Variable |    2     |    2     | Varibale |
// +----+-----+-------+------+----------+----------+----------+----------+

func Connection(conn net.Conn) (*ConnRequest, error) {
	offset := 0
	req := &ConnRequest{}
	// 读取协商数据
	buf := make([]byte, 512)
	if _, err := io.ReadFull(conn, buf[:5]); err != nil { // addition 1 byte for DST.ADDR.TYPE
		return nil, err
	}

	if buf[offset] != Version {
		log.Printf("bad version: %d", buf[offset])
		return nil, ErrBadVersion
	}
	offset += 1

	req.Cmd = buf[offset]
	offset += 1
	log.Printf("Cmd: %d", req.Cmd)

	req.Resverd = buf[offset]
	offset += 1

	req.Addr = &Addr{}
	req.Addr.Type = buf[offset]
	offset += 1
	log.Printf("AddrType: %d", req.Addr.Type)

	addrLen := 0
	switch req.Addr.Type {
	case 0x01: // IPv4
		addrLen = 3
	case 0x03: // Domain
		addrLen = 7 + int(buf[offset])
	case 0x04: // IPv6
		addrLen = 15
	default:
		return nil, errors.New("unsupported address type")
	}
	offset += 1

	if _, err := io.ReadFull(conn, buf[offset:offset+addrLen+2+2]); err != nil {
		return nil, err
	}

	if req.Addr.Type == 0x03 {
		// byte to string
		req.Addr.Host = string(buf[offset : offset+addrLen])
	} else if req.Addr.Type == 0x01 {
		// byte to net.IP
		addr := net.IP(buf[4 : offset+addrLen])
		req.Addr.Host = addr.String()
	}

	switch req.Addr.Type {
	case 0x01:
		addr := net.IP(buf[4 : offset+addrLen])
		req.Addr.Host = addr.String()
	case 0x03:
		req.Addr.Host = string(buf[offset : offset+addrLen])
	case 0x04:
		addr := net.IP(buf[4 : offset+addrLen])
		req.Addr.Host = addr.String()
	}

	offset += addrLen
	log.Printf("Addr: %s", req.Addr.Host)

	req.Addr.Port = uint16(buf[offset])<<8 | uint16(buf[offset+1])
	offset += 2
	log.Printf("Port: %d", req.Addr.Port)

	req.ExtraLen = uint16(buf[offset])<<8 | uint16(buf[offset+1])
	offset += 2
	log.Printf("ExtraLen: %d", req.ExtraLen)

	if req.ExtraLen > 0 {
		if _, err := io.ReadFull(conn, buf[offset:offset+int(req.ExtraLen)]); err != nil {
			return nil, err
		}
		// offset += int(req.ExtraLen)
	}

	base64str := string(buf[offset : offset+int(req.ExtraLen)])
	log.Printf("Base64 len: %d, Base64: %s", len(base64str), base64str)

	str, err := base64.StdEncoding.DecodeString(base64str)
	if err != nil {
		return nil, err
	}

	log.Printf("ExtraData: {%s}", string(str))

	req.ExtraData = ExtData{}
	if err := json.Unmarshal(str, &req.ExtraData); err != nil {
		return nil, err
	}

	log.Printf("Cmd: %d, Addr: %s:%d, ExtraData: %v", req.Cmd, req.Addr.Host, req.Addr.Port, req.ExtraData)

	return req, nil
}
