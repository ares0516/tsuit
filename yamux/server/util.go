package main

import "fmt"

func allocAddr() string {
	for i := 1; i < 255; i++ {
		addr := fmt.Sprintf("10.0.0.%d", i)
		if !_manager.IsExist(addr) {
			return addr
		}
	}
	return ""
}
