package main

import (
	_ "embed"
	"fmt"
	"net/http"
)

//go:embed ws.html
var indexHTML string

func main() {
	mux := http.NewServeMux()
	hub := NewHub()

	go hub.run()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, indexHTML)
	})
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWS(hub, w, r)
	})

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		fmt.Println(err)
	}
}
