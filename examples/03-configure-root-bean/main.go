package main

import (
	"fmt"
	"net/http"

	"github.com/go-spring/spring-core/gs"
)

type GreetingRoot struct {
	Greeting string `value:"${demo.greeting:=Hello}" expr:"$ != ''"`
	Audience string `value:"${demo.audience:=Go-Spring}" expr:"$ != ''"`
}

// Hello is an HTTP handler.
func (g *GreetingRoot) Hello(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintf(w, "%s, %s!\n", g.Greeting, g.Audience)
}

func main() {
	root := &GreetingRoot{}

	// Register HTTP handler using the bean's method
	http.HandleFunc("/hello", root.Hello)

	// Configure the application
	gs.Configure(func(app gs.App) {
		app.Root(root)
	}).Run()
}

// Start it with `go run main.go`.
// Then try `curl http://127.0.0.1:9090/hello`.
// It should return `Hello, Go-Spring!`.
// Those values come from the root bean fields bound by Go-Spring.
// Press `Ctrl+C` when you want to stop it.
