package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from backend 2 (port 8082)\n")
	})
	fmt.Println("Starting server on :8082")
	http.ListenAndServe(":8082", nil)
}
