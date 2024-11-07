package main

import (
	"encoding/json"
	"log"
	"net"
)

type Request struct {
	Action string `json:"action"`
}

type Reply struct {
	Msg string `json:"msg"`
}

func handlerCmdGatewaySate(cli net.Conn) error {
	cli.Write([]byte{Version, Success, 0x00})

	errChan := make(chan error, 1)
	go func() {
		for {
			buf := make([]byte, 1024)
			n, err := cli.Read(buf)
			if err != nil {
				errChan <- err
				return
			}

			req := &Request{}
			if err := json.Unmarshal(buf[:n], req); err != nil {
				continue
			}

			switch req.Action {
			case "heart":
				_, err = cli.Write([]byte("{\"state\":\"0\"}"))
			case "onlineuser":
				_, err = cli.Write([]byte("{\"msg\":\"1\",\"state\":\"0\"}"))
			case "traffic":
				_, err = cli.Write([]byte("{\"msg\":\"100,100\",\"state\":\"0\"}"))
			}

			if err != nil {
				errChan <- err
				return
			}
		}
	}()

	err := <-errChan

	log.Printf("网关状态处理失败: %v, Addr: %v", err, cli.RemoteAddr())

	return nil
}
