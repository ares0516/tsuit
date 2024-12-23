package main

import (
	"fmt"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	// Write "Hello World" as the response
	fmt.Fprintf(w, "Hello World")
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
