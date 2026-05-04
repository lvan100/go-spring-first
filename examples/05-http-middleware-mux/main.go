package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-spring/spring-core/gs"
)

type GreetingConfig struct {
	Greeting string `value:"${greeting:=Hello}" expr:"$ != ''"`
	Audience string `value:"${audience:=Go-Spring}" expr:"$ != ''"`
}

type Controller struct {
	cfg GreetingConfig
}

// NewController creates a new controller with the given configuration.
// The configuration is injected by Go-Spring.
func NewController(cfg GreetingConfig) *Controller {
	return &Controller{cfg: cfg}
}

// Hello is an HTTP handler.
func (c *Controller) Hello(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintf(w, "%s, %s!\n", c.cfg.Greeting, c.cfg.Audience)
}

// NewHTTPMux creates a new HTTP mux with the given controller.
func NewHTTPMux(c *Controller) *gs.HttpServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", c.Hello)
	// Add logging middleware
	return &gs.HttpServeMux{Handler: logging(mux)}
}

// logging is a middleware that logs the request method, path, and elapsed time.
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("method=%s path=%s elapsed=%s",
			r.Method, r.URL.Path, time.Since(start))
	})
}

func init() {
	gs.Provide(NewController, gs.TagArg("${demo}"))
	gs.Provide(NewHTTPMux)
}

func main() {
	gs.Run()
}

// Start it with `go run main.go`.
// Then try `curl -i http://127.0.0.1:9090/hello`.
// It should return `Hello with middleware, custom mux!`.
// The console should also print the method, path, and elapsed time from the middleware.
// Press `Ctrl+C` when you want to stop it.
