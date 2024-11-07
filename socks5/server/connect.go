package main

import (
	"io"
	"log"
	"net"
)

func handlerCmdConnect(cli net.Conn, req *ConnRequest) error {
	cli.Write([]byte{Version, Success, 0x00})

	dstAddr := req.Addr.Host + ":" + string(req.Addr.Port)

	dstCli, err := net.Dial("tcp", dstAddr)
	if err != nil {
		cli.Write([]byte{Version, Unreachable, 0x00})
		return err
	}
	defer dstCli.Close()

	cli.Write([]byte{Version, Success, 0x00})

	err = pipe(cli, dstCli)

	if err != nil {
		log.Printf("proxy forward: %v", err)
	}
	return err
}

func pipe(src, dst net.Conn) error {
	errChan := make(chan error, 1)

	go func() {
		_, err := io.Copy(src, dst)
		if err != nil {
			errChan <- err
		}
	}()

	go func() {
		_, err := io.Copy(dst, src)
		if err != nil {
			errChan <- err
		}
	}()

	err := <-errChan

	src.Close()
	dst.Close()
	return err
}
