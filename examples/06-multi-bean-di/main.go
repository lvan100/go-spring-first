package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-spring/spring-core/gs"
)

type GreetingService struct {
	Greeting string `value:"${demo.greeting:=Hello}" expr:"$ != ''"`
}

func NewGreetingService() *GreetingService {
	return &GreetingService{}
}

func (s *GreetingService) Message(audience string) string {
	return fmt.Sprintf("%s, %s!", s.Greeting, audience)
}

type Controller struct {
	service  *GreetingService
	Audience string `value:"${demo.audience:=Go-Spring}" expr:"$ != ''"`
}

// NewController creates a new controller with the given service.
// The service is injected by Go-Spring.
func NewController(service *GreetingService) *Controller {
	return &Controller{service: service}
}

// Hello is an HTTP handler.
func (c *Controller) Hello(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintln(w, c.service.Message(c.Audience))
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
	gs.Provide(NewGreetingService)
	gs.Provide(NewController)
	gs.Provide(NewHTTPMux)
}

func main() {
	gs.Run()
}

// Start it with `go run main.go`.
// Then try `curl http://127.0.0.1:9090/hello`.
// It should return `Hello from service, controller config!`.
// That response is built by the service and called through the controller.
// Press `Ctrl+C` when you want to stop it.
