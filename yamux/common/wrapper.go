// common/common.go
package common

import (
	"fmt"

	"github.com/armon/go-socks5"
)

func CommonFunction() {
	fmt.Println("This is a common function.")
}

func NewSimpleSocksProxyServer() (*socks5.Server, error) {
	conf := &socks5.Config{}
	server, err := socks5.New(conf)
	if err != nil {
		return nil, err
	}

	return server, nil
}
