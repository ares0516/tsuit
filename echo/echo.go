package main

import (
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

func main() {
	// Register the handler function for the root path "/"
	http.HandleFunc("/", handler)

	// Start the server on port 8080
	fmt.Println("Server starting on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}
