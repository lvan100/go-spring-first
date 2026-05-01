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

func NewController(service *GreetingService) *Controller {
	return &Controller{service: service}
}

func (c *Controller) Hello(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintln(w, c.service.Message(c.Audience))
}

func NewHTTPMux(c *Controller) *gs.HttpServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", c.Hello)
	return &gs.HttpServeMux{Handler: requestID(logging(mux))}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r)
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("method=%s path=%s status=%d elapsed=%s request_id=%s",
			r.Method, r.URL.Path, rec.status, time.Since(start), w.Header().Get("X-Request-ID"))
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
