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

// Start with `go run main.go`, then try `curl http://127.0.0.1:9090/hello`.
// It should use `./conf/app.properties` and say `config file`.
// Restart with `GS_DEMO_AUDIENCE="env var" go run main.go`; the same curl should say `env var`.
// Restart with `go run main.go -Ddemo.audience="cmd arg"`; the same curl should say `cmd arg`.
// Press `Ctrl+C` between runs.
