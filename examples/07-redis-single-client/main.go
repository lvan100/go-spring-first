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

func (c *RedisClient) Ping(context.Context) error {
	log.Printf("redis ping addr=%s", c.cfg.Addr)
	return nil
}

type GreetingService struct {
	redis    *RedisClient
	Greeting string `value:"${demo.greeting:=Hello}" expr:"$ != ''"`
}

// NewGreetingService creates a new greeting service with the given Redis client.
// The Redis client is injected by Go-Spring.
func NewGreetingService(redis *RedisClient) *GreetingService {
	return &GreetingService{redis: redis}
}

func (s *GreetingService) Message(ctx context.Context, audience string) string {
	_ = s.redis.Ping(ctx)
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
	_, _ = fmt.Fprintln(w, c.service.Message(r.Context(), c.Audience))
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
	gs.Provide(NewRedisClient, gs.TagArg("${spring.go-redis}")).Destroy(CloseRedis)
	gs.Provide(NewGreetingService)
	gs.Provide(NewController)
	gs.Provide(NewHTTPMux)
}

func main() {
	gs.Run()
}

// Start it with `go run main.go`.
// The console should print `create redis client addr=127.0.0.1:6379`.
// Then try `curl http://127.0.0.1:9090/hello`.
// It should return `Hello with Redis, single client!`.
// The console should print a redis ping, but this sample does not connect to real Redis.
// Press `Ctrl+C` when you want to stop it.
