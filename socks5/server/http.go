package main

import (
	"fmt"
	"log"
	"net/http"
)

func http_start() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello")
	})

	addr := "0.0.0.0:8080"
	log.Println("HTTP 服务器正在监听 " + addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("无法启动 HTTP 服务器: %v", err)
	}
}
