package main

import (
	"flag"
	"fmt"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	// 设置Content-Type为text/html
	w.Header().Set("Content-Type", "text/html")

	// 返回HTML内容
	htmlContent := `
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>HTML Page</title>
    </head>
    <body>
        <h1>Hello, World!</h1>
        <p>This is a simple HTML page.</p>
    </body>
    </html>
    `
	w.Write([]byte(htmlContent))
}

var port string

func main() {

	flag.StringVar(&port, "port", "8000", "The port to listen on")

	// Register the handler function for the root path "/"
	http.HandleFunc("/", handler)

	// Start the server on port 8000
	fmt.Println("Server starting on port 8000...")
	addr := fmt.Sprintf(":%s", port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}
