package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-spring/log"
	"github.com/go-spring/spring-core/gs"
)

type requestIDKey struct{}

var (
	tagBizGreeting = log.RegisterBizTag("greeting", "serve")
	tagHTTPRequest = log.RegisterRPCTag("http", "request")
)

type GreetingService struct {
	Greeting string `value:"${demo.greeting:=Hello}" expr:"$ != ''"`
}

func NewGreetingService() *GreetingService {
	return &GreetingService{}
}

func (s *GreetingService) Summary(ctx context.Context) string {
	log.Info(ctx, tagBizGreeting,
		log.String("greeting", s.Greeting),
		log.Msg("building greeting"),
	)
	return s.Greeting + ", structured logs!"
}

type Controller struct {
	service *GreetingService
}

// NewController creates a new controller with the given service.
// The service is injected by Go-Spring.
func NewController(service *GreetingService) *Controller {
	return &Controller{service: service}
}

// Hello is an HTTP handler.
func (c *Controller) Hello(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintln(w, c.service.Summary(r.Context()))
}

// NewHTTPMux creates a new HTTP mux with the given controller.
// The controller is injected by Go-Spring.
func NewHTTPMux(c *Controller) *gs.HttpServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", c.Hello)
	// Add request ID and logging middleware
	return &gs.HttpServeMux{Handler: requestID(logging(mux))}
}

// requestID is a middleware that adds a request ID to the request context.
// The request ID is extracted from the X-Request-ID header, or generated if not present.
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// logging is a middleware that logs HTTP requests.
// It logs the request method, path, elapsed time, and greeting field.
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Info(r.Context(), tagHTTPRequest,
			log.String("method", r.Method),
			log.String("path", r.URL.Path),
			log.String("elapsed", time.Since(start).String()),
			log.Msg("http request completed"),
		)
	})
}

func init() {
	// Extract request ID from the context and add it to the log fields.
	log.FieldsFromContext = func(ctx context.Context) []log.Field {
		id, ok := ctx.Value(requestIDKey{}).(string)
		if !ok || id == "" {
			return nil
		}
		return []log.Field{log.String("request_id", id)}
	}

	gs.Provide(NewGreetingService)
	gs.Provide(NewController)
	gs.Provide(NewHTTPMux)
}

func main() {
	gs.Run()
}

// Start it with `go run main.go`.
// Then try `curl -H "X-Request-ID: demo-1" http://127.0.0.1:9090/hello`.
// It should return `Hello with logging, structured logs!`.
// The console should print JSON logs with the tag, request_id, method, path, elapsed time, and greeting field.
// Press `Ctrl+C` when you want to stop it.
