package common

import (
	"bytes"
	"io"
	"net"

	"gvisor.dev/gvisor/pkg/errors"
)

var token = []byte("in the pipe, five by five")

func PipeAuth(conn net.Conn) error {
	auth := &bytes.Buffer{}
	len := len(token)
	auth.WriteByte(byte(len))
	auth.Write(token)

	// 发送认证数据
	_, err := conn.Write(auth.Bytes())
	if err != nil {
		return err
	}

	// 读取响应
	response := make([]byte, 4)
	if _, err := io.ReadFull(conn, response); err != nil {
		return err
	}

	// 检查响应
	if string(response) == "succ" {
		return nil
	}
	return errors.New("authentication failed")
}

func PipeCheck(conn net.Conn) error {
	// 读取token长度 (1字节)
	lenBuf := make([]byte, 1)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return err
	}

	// 获取token长度
	tokenLen := int(lenBuf[0])

	// 读取token内容
	recvToken := make([]byte, tokenLen)
	if _, err := io.ReadFull(conn, recvToken); err != nil {
		return err
	}

	// 验证token是否匹配
	if !bytes.Equal(recvToken, token) {
		// 发送失败响应
		if _, err := conn.Write([]byte("fail")); err != nil {
			return err
		}
		return errors.New("token mismatch")
	}

	// 发送成功响应
	if _, err := conn.Write([]byte("succ")); err != nil {
		return err
	}
	return nil
}
