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
	tagHTTPRequest = log.RegisterRPCTag("http", "request")
)

type RedisPinger interface {
	Ping(context.Context) error
}

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
	log.Info(context.Background(), log.TagBizDef,
		log.String("addr", cfg.Addr),
		log.Msg("create redis client"),
	)
	return &RedisClient{cfg: cfg}, nil
}

// CloseRedis closes the Redis client.
func CloseRedis(*RedisClient) error {
	return nil
}

func (c *RedisClient) Ping(ctx context.Context) error {
	log.Info(ctx, log.TagBizDef, log.String("addr", c.cfg.Addr), log.Msg("redis ping"))
	return nil
}

type GreetingService struct {
	redis    RedisPinger
	Greeting string `value:"${demo.greeting:=Hello}" expr:"$ != ''"`
}

func NewGreetingService(redis RedisPinger) *GreetingService {
	return &GreetingService{redis: redis}
}

func (s *GreetingService) Message(ctx context.Context, audience string) string {
	if err := s.redis.Ping(ctx); err != nil {
		log.Warn(ctx, log.TagBizDef, log.String("err", err.Error()), log.Msg("redis ping failed"))
	} else {
		log.Info(ctx, log.TagBizDef, log.String("audience", audience), log.Msg("greeting built"))
	}
	return fmt.Sprintf("%s, %s!", s.Greeting, audience)
}

type Controller struct {
	service  *GreetingService
	Audience string `value:"${demo.audience:=Go-Spring}" expr:"$ != ''"`
}

// NewController creates a new controller with the given greeting service.
// The greeting service is injected by Go-Spring.
func NewController(service *GreetingService) *Controller {
	return &Controller{service: service}
}

// Hello is an HTTP handler.
func (c *Controller) Hello(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintln(w, c.service.Message(r.Context(), c.Audience))
}

// NewHTTPMux creates a new HTTP mux with the given controller.
// The controller is injected by Go-Spring.
func NewHTTPMux(c *Controller) *gs.HttpServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", c.Hello)
	// Wrap the mux with request ID middleware and logging middleware.
	return &gs.HttpServeMux{Handler: requestID(logging(mux))}
}

// requestID is a middleware that adds a request ID to the context.
// If the request ID is not in the request header, it is generated.
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
// The request ID is extracted from the context and added to the log fields.
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

	// Provide Redis client and export it as RedisPinger.
	gs.Provide(NewRedisClient, gs.TagArg("${spring.go-redis}")).
		Condition(gs.OnProperty("spring.go-redis.addr")).
		Destroy(CloseRedis).
		Export(gs.As[RedisPinger]())

	gs.Provide(NewGreetingService)
	gs.Provide(NewController)
	gs.Provide(NewHTTPMux)
}

func main() {
	gs.Run()
}

// Run `go test ./...` first.
// It should pass without a real Redis server or a real HTTP server.
// The tests use a fake Redis pinger and Go-Spring's test container.
// You can still start it with `go run main.go`.
// Then try `curl http://127.0.0.1:9090/hello`.
// It should return `Hello under test, testable controller!`.
// Press `Ctrl+C` when you want to stop it.
