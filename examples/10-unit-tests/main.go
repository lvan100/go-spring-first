package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	gslog "github.com/go-spring/log"
	"github.com/go-spring/spring-core/gs"
)

type requestIDKey struct{}

var (
	tagBizGreeting = gslog.RegisterBizTag("greeting", "serve")
	tagHTTP        = gslog.RegisterRPCTag("http", "request")
	tagRedis       = gslog.RegisterRPCTag("redis", "ping")
)

type RedisPinger interface {
	Ping(context.Context) error
}

type RedisConfig struct {
	Addr     string `value:"${addr}" expr:"$ != ''"`
	Password string `value:"${password:=}"`
}

type RedisClient struct {
	cfg     RedisConfig
	timeout time.Duration
}

func NewRedisClient(cfg RedisConfig) (*RedisClient, error) {
	return &RedisClient{cfg: cfg, timeout: 500 * time.Millisecond}, nil
}

func CloseRedis(*RedisClient) error {
	return nil
}

func (c *RedisClient) Ping(ctx context.Context) error {
	line, err := c.do(ctx, "PING")
	if err != nil {
		gslog.Warn(ctx, tagRedis,
			gslog.String("addr", c.cfg.Addr),
			gslog.String("err", err.Error()),
			gslog.Msg("redis ping failed"),
		)
		return err
	}
	gslog.Info(ctx, tagRedis, gslog.String("addr", c.cfg.Addr), gslog.Msg("redis ping ok"))
	if strings.TrimSpace(line) != "+PONG" {
		return fmt.Errorf("unexpected redis response %q", strings.TrimSpace(line))
	}
	return nil
}

func (c *RedisClient) do(ctx context.Context, args ...string) (string, error) {
	d := net.Dialer{Timeout: c.timeout}
	conn, err := d.DialContext(ctx, "tcp", c.cfg.Addr)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(c.timeout))

	reader := bufio.NewReader(conn)
	if c.cfg.Password != "" {
		if err := writeRESP(conn, "AUTH", c.cfg.Password); err != nil {
			return "", err
		}
		if line, err := reader.ReadString('\n'); err != nil {
			return "", err
		} else if strings.HasPrefix(line, "-") {
			return "", errors.New(strings.TrimSpace(line))
		}
	}
	if err := writeRESP(conn, args...); err != nil {
		return "", err
	}
	return reader.ReadString('\n')
}

func writeRESP(w io.Writer, args ...string) error {
	if _, err := fmt.Fprintf(w, "*%d\r\n", len(args)); err != nil {
		return err
	}
	for _, arg := range args {
		if _, err := fmt.Fprintf(w, "$%d\r\n%s\r\n", len(arg), arg); err != nil {
			return err
		}
	}
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
		gslog.Warn(ctx, tagBizGreeting, gslog.String("err", err.Error()), gslog.Msg("using degraded response"))
		return fmt.Sprintf("%s, %s! redis unavailable", s.Greeting, audience)
	}
	gslog.Info(ctx, tagBizGreeting, gslog.String("audience", audience), gslog.Msg("greeting built"))
	return fmt.Sprintf("%s, %s! redis=PONG", s.Greeting, audience)
}

type Controller struct {
	service  *GreetingService
	Audience string `value:"${demo.audience:=Go-Spring}" expr:"$ != ''"`
}

func NewController(service *GreetingService) *Controller {
	return &Controller{service: service}
}

func (c *Controller) Hello(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintln(w, c.service.Message(r.Context(), c.Audience))
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
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		gslog.Info(r.Context(), tagHTTP,
			gslog.String("method", r.Method),
			gslog.String("path", r.URL.Path),
			gslog.Int("status", rec.status),
			gslog.String("elapsed", time.Since(start).String()),
			gslog.Msg("http request completed"),
		)
	})
}

func configureLogging() {
	err := gslog.RefreshConfig(map[string]string{
		"appender.console.type":                     "ConsoleAppender",
		"appender.console.layout.type":              "JSONLayout",
		"appender.console.layout.fileLineMaxLength": "80",
		"logger.root.type":                          "SyncLogger",
		"logger.root.level":                         "INFO",
		"logger.root.appenderRef.ref":               "console",
		"logger.biz.type":                           "SyncLogger",
		"logger.biz.level":                          "INFO",
		"logger.biz.tag":                            "_biz_greeting_*",
		"logger.biz.appenderRef.ref":                "console",
		"logger.http.type":                          "SyncLogger",
		"logger.http.level":                         "INFO",
		"logger.http.tag":                           "_rpc_http_*",
		"logger.http.appenderRef.ref":               "console",
		"logger.redis.type":                         "SyncLogger",
		"logger.redis.level":                        "INFO",
		"logger.redis.tag":                          "_rpc_redis_*",
		"logger.redis.appenderRef.ref":              "console",
	})
	if err != nil {
		panic(err)
	}
}

func init() {
	configureLogging()

	gslog.FieldsFromContext = func(ctx context.Context) []gslog.Field {
		id, ok := ctx.Value(requestIDKey{}).(string)
		if !ok || id == "" {
			return nil
		}
		return []gslog.Field{gslog.String("request_id", id)}
	}

	gs.Provide(NewRedisClient, gs.TagArg("${spring.go-redis}")).
		Condition(gs.OnProperty("spring.go-redis.addr")).
		Destroy(CloseRedis).
		Name("__default__").
		Export(gs.As[RedisPinger]())
	gs.Provide(NewGreetingService)
	gs.Provide(NewController)
	gs.Provide(NewHTTPMux)
}

func main() {
	gs.Run()
}
