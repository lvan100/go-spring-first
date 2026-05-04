package main

import (
	"net/http"

	"github.com/go-spring/spring-core/gs"
)

func main() {
	// Register handler using standard http.HandleFunc
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello from net/http\n"))
	})

	gs.Run()
}

// Start it with `go run main.go`.
// Then try `curl http://127.0.0.1:9090/hello`.
// It should return `hello from net/http`.
// The handler is just the standard `http.HandleFunc`, served by Go-Spring.
// Press `Ctrl+C` when you want to stop it.
