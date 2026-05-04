package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-spring/spring-core/gs"
)

type RedisConfig struct {
	Addr     string `value:"${addr}" expr:"$ != ''"`
	Password string `value:"${password:=}"`
}

type RedisClient struct {
	cfg RedisConfig
}

// NewRedisClient creates a new Redis client with the given configuration.
// The configuration is injected by Go-Spring.
func NewRedisClient(cfg RedisConfig) (*RedisClient, error) {
	log.Printf("create redis client addr=%s", cfg.Addr)
	return &RedisClient{cfg: cfg}, nil
}

// CloseRedis closes the Redis client.
func CloseRedis(*RedisClient) error {
	return nil
}

func (c *RedisClient) Addr() string {
	return c.cfg.Addr
}

func (c *RedisClient) Ping(context.Context) error {
	log.Printf("redis ping addr=%s", c.cfg.Addr)
	return nil
}

type GreetingService struct {
	Client   *RedisClient `autowire:"__default__?"`
	Greeting string       `value:"${demo.greeting:=Hello}" expr:"$ != ''"`
}

func NewGreetingService() *GreetingService {
	return &GreetingService{}
}

func (s *GreetingService) Summary(ctx context.Context) string {
	_ = s.Client.Ping(ctx)
	return s.Greeting + ", conditional clients!"
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
	// Provide Redis client
	gs.Provide(NewRedisClient, gs.TagArg("${spring.go-redis}")).
		Condition(gs.OnProperty("spring.go-redis.addr")).
		Destroy(CloseRedis).
		Name("__default__")

	// Provide multiple Redis clients
	gs.Group("${spring.go-redis.instances}", NewRedisClient, CloseRedis)

	gs.Provide(NewGreetingService)
	gs.Provide(NewController)
	gs.Provide(NewHTTPMux)
}

func main() {
	gs.Run()
}

// Start it with `go run main.go`.
// The console should show clients created for `6379`, `6380`, and `6381`.
// Then try `curl http://127.0.0.1:9090/hello`.
// It should return `Hello with conditional Redis, conditional clients!`.
// The ping log shows which named client the service actually uses.
// Press `Ctrl+C` when you want to stop it.
