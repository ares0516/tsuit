package main

import (
	"errors"
	"io"
	"log"
	"net"
	"strconv"

	"test.com/server/bufpool"
)

func handlerCmdConnect(cli net.Conn, req *ConnRequest) error {
	cli.Write([]byte{Version, Success, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	dstAddr := req.Addr.Host + ":" + strconv.Itoa(int(req.Addr.Port))

	log.Printf("proxy connect: %v", dstAddr)

	dstCli, err := net.Dial("tcp", dstAddr)
	if err != nil {
		cli.Write([]byte{Version, Unreachable, 0x00})
		return err
	}
	defer dstCli.Close()

	log.Printf("proxy connect success: %v", dstAddr)

	cli.Write([]byte{Version, Success, 0x00})

	pipe(cli, dstCli)

	return err
}

const (
	BufSize = 20 << 10
)

func exchangeBuffer2(src, dst net.Conn) (err error) {
	buf := bufpool.Get(32768)
	defer bufpool.Put(buf)

	for {
		nRead, errRead := src.Read(buf)
		if nRead > 0 {
			nWrite, errWrite := dst.Write(buf[:nRead])
			if nWrite < 0 || nRead < nWrite {
				nWrite = 0
				if errWrite == nil {
					errWrite = errors.New("invalid write result")
				}
				if errWrite != nil {
					err = errWrite
					break
				}
				if nRead != nWrite {
					err = io.ErrShortWrite
					break
				}
			}
			if errRead != nil {
				if errRead != io.EOF {
					err = errRead
				}
				break
			}
		}

	}
	log.Printf("exchangeBuffer err: %v", err)
	bufpool.Put(buf)
	return err
}

func exchangeBuffer1(src, dst net.Conn) (err error) {
	buf := bufpool.Get(BufSize)
	defer bufpool.Put(buf)

	for {
		nRead, errRead := src.Read(buf)
		if errRead != nil {
			if errRead != io.EOF {
				err = errRead
			}
			log.Printf("read data 1: %v , %d , %v", err, nRead, errRead)
			break
		}

		if nRead > 0 {
			nWrite, errWrite := dst.Write(buf[:nRead])
			if errWrite != nil {
				err = errWrite
				log.Printf("write data 1: %v", err)
				break
			}
			if nWrite != nRead {
				err = io.ErrShortWrite
				log.Printf("write data 2: %v", err)
				break
			}
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("read data 2: %v", err)
			}
			break
		}
	}
	log.Printf("exchangeBuffer err: %v", err)
	bufpool.Put(buf)
	return err
}

func pipe(src, dst net.Conn) error {
	errChan := make(chan error, 1)
	log.Printf("pipe %v <--> %v", src.RemoteAddr(), dst.RemoteAddr())
	go func() {
		err := exchangeBuffer2(src, dst)
		errChan <- err
	}()

	go func() {
		err := exchangeBuffer2(dst, src)
		errChan <- err

	}()

	err := <-errChan

	log.Printf("pipe err: %v", err)

	src.Close()
	dst.Close()
	return err
}
