package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from backend 1 (port 8081)\n")
	})
	fmt.Println("Starting server on :8081")
	http.ListenAndServe(":8081", nil)
}
