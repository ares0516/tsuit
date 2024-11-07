package main

func main() {
	go socks_start(true)
	go http_start()

	select {}
}
