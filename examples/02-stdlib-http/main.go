package main

import (
	"net/http"

	"github.com/go-spring/spring-core/gs"
)

func main() {
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello from net/http\n"))
	})

	gs.Run()
}
