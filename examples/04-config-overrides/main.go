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

func (g *GreetingRoot) Hello(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintf(w, "%s, %s!\n", g.Greeting, g.Audience)
}

func main() {
	root := &GreetingRoot{}
	http.HandleFunc("/hello", root.Hello)

	gs.Configure(func(app gs.App) {
		app.Root(root)
	}).Run()
}
