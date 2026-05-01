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

func NewController(cfg GreetingConfig) *Controller {
	return &Controller{cfg: cfg}
}

func (c *Controller) Hello(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintf(w, "%s, %s!\n", c.cfg.Greeting, c.cfg.Audience)
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
	gs.Provide(NewController, gs.TagArg("${demo}"))
	gs.Provide(NewHTTPMux)
}

func main() {
	gs.Run()
}
