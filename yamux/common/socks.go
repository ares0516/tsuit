package common

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"syscall"
	"unicode"
)

const Zero = 0x0
const (
	// Version 协议版本
	Version = 0x05
)

// 认证方法
const (
	// NoAuthenticationRequired 不需要认证
	NoAuthenticationRequired = 0x00
	// AccountPasswordAuthentication 账号密码认证
	AccountPasswordAuthentication = 0x02
)

// 命令
const (
	// Connect 连接上游服务器
	Connect = iota + 1
	//Bind 绑定请求
	Bind
	// UDP 转发
	UDP
)

// 目标地址类型
const (
	//IPV4 DST.ADDR部分4字节长度
	IPV4 = 0x01
	// Com 域名
	Com = 0x03
	// IPV6 16个字节长度
	IPV6 = 0x04
)

type AuthPackage struct {
	methodsCount uint8
	methods      []byte
}

func (t *AuthPackage) toData() []byte {
	return append([]byte{Version, t.methodsCount}, t.methods...)
}
func (t *AuthPackage) addMethod(methods ...uint8) {
	t.methods = append(t.methods, methods...)
	t.methodsCount += uint8(len(methods))
}

type ClientRequest struct {
	command     uint8
	RSV         uint8
	addressType uint8
	addr        []byte
}

func Auth(conn net.Conn) error {
	//组织发送支持的认证方法
	authPackage := AuthPackage{}

	//免密
	authPackage.addMethod(NoAuthenticationRequired)
	_, err := conn.Write(authPackage.toData())
	if err != nil {
		return err
	}
	data := make([]byte, 2)
	l, err := conn.Read(data)
	if err != nil {
		return err
	}
	if l != 2 {
		return errors.New("返回数据有误非两个字节")
	}
	if data[0] != Version {
		return errors.New("当前协议Socks5与服务端协议不匹配")
	}

	switch data[1] {
	case NoAuthenticationRequired:
		return nil
	}

	return errors.New("非认证模式认证失败")
}

// 向服务器发送连接请求
func Requisition(conn net.Conn, host string, port uint16, cmd uint8) error {
	Type, addr, err := addressResolution(host)
	if err != nil {
		return err
	}
	buffer := bytes.Buffer{}
	buffer.Write([]byte{Version, cmd, Zero, Type})
	buffer.Write(addr)
	//写入端口
	buffer.Write(portToBytes(port))
	_, err = conn.Write(buffer.Bytes())
	if err != nil {
		return err
	}

	// 读取响应，此处粗暴多读一点
	resp := make([]byte, 10)
	_, err = io.ReadFull(conn, resp)
	if err != nil {
		return err
	}

	// 检查响应是否正确
	if resp[0] != Version {
		return errors.New("服务器响应的协议版本与期望不符")
	}
	if resp[1] != 0x00 {
		return errors.New("服务器拒绝连接")
	}

	return nil
}

func addressResolution(host string) (uint8, []byte, error) {
	var Type uint8
	addr := make([]byte, 0)
	if strings.Contains(host, ":") {
		Type = IPV6
		//ipv6 地址转16位uint
		if strings.Contains(host, "[") {
			host = host[1 : len(host)-1]
		}
		split := strings.Split(host, ":")
		for _, s := range split {
			if s == "" {
				addr = append(addr, Zero)
				continue
			}
			num, err := strconv.ParseUint(s, 16, 16)
			if err != nil {
				return Zero, nil, err
			}
			bytes := make([]byte, 2)
			binary.BigEndian.PutUint16(bytes, uint16(num))
			addr = append(addr, bytes...)
		}
	}
	runes := []rune(host)
	if Type == Zero && unicode.IsNumber(runes[len(runes)-1]) {
		Type = IPV4
		split := strings.Split(host, ".")
		for _, s := range split {
			num, err := strconv.ParseUint(s, 10, 16)
			if err != nil {
				return Zero, nil, err
			}
			addr = append(addr, uint8(num))
		}
	}
	if Type == Zero && unicode.IsLetter(runes[len(runes)-1]) {
		Type = Com
		addr = append(addr, uint8(len(host)))
		addr = append(addr, []byte(host)...)
	}
	if Type == Zero {
		return Zero, nil, errors.New("地址类型错误")
	}
	return Type, addr, nil
}

// 从[]byte 数据中解析主机地址和端口
func addressResolutionFormByteArray(ipdata []byte, Type uint8) (string, error) {
	if len(ipdata) < 6 || Type == Zero {
		return "", errors.New(fmt.Sprintf("解析地址数据有误,%v %x", ipdata, Type))
	}
	addr := ""
	var portBytes []byte
	switch Type {
	case IPV4:
		for i, b := range ipdata[:4] {
			addr += strconv.Itoa(int(b))
			if i != 3 {
				addr += "."
			}
		}
		portBytes = ipdata[4:6]
	case IPV6:
		if len(ipdata) < 18 {
			return "", errors.New("数据长度不足18字节")
		}
		for i := 0; i < 16; i += 2 {
			u := binary.BigEndian.Uint16(ipdata[i : i+2])
			s := strconv.FormatUint(uint64(u), 16)
			addr += s
			if i != 14 {
				addr += ":"
			}
		}
		portBytes = ipdata[16:18]
	case Com:
		l := ipdata[0]
		if int(l)+3 < len(ipdata) {
			return "", errors.New("数据长度不足")
		}
		bytes := ipdata[1 : int(l)+1]
		addr += string(bytes)
		portBytes = ipdata[int(l)+1 : int(l)+3]
	}
	addr += ":" + strconv.Itoa(int(binary.BigEndian.Uint16(portBytes)))
	return addr, nil
}

func portToBytes(port uint16) []byte {
	//写入端口
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, port)
	return buf
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
