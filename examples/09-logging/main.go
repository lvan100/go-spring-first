package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
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

func (c *RedisClient) Addr() string {
	return c.cfg.Addr
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
	Default  *RedisClient            `autowire:"__default__?"`
	Clients  map[string]*RedisClient `autowire:"?"`
	Greeting string                  `value:"${demo.greeting:=Hello}" expr:"$ != ''"`
}

func NewGreetingService() *GreetingService {
	return &GreetingService{}
}

func (s *GreetingService) Summary(ctx context.Context) string {
	lines := []string{s.Greeting + ", structured logs!"}
	gslog.Info(ctx, tagBizGreeting, gslog.Int("client_count", len(s.Clients)), gslog.Msg("building greeting"))

	if s.Default == nil {
		lines = append(lines, "default: disabled")
	} else if err := s.Default.Ping(ctx); err != nil {
		lines = append(lines, fmt.Sprintf("default(%s): unavailable: %v", s.Default.Addr(), err))
	} else {
		lines = append(lines, fmt.Sprintf("default(%s): PONG", s.Default.Addr()))
	}

	names := make([]string, 0, len(s.Clients))
	for name := range s.Clients {
		if name != "__default__" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		lines = append(lines, "instances: none")
	}
	for _, name := range names {
		client := s.Clients[name]
		if err := client.Ping(ctx); err != nil {
			lines = append(lines, fmt.Sprintf("%s(%s): unavailable: %v", name, client.Addr(), err))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s(%s): PONG", name, client.Addr()))
	}
	return strings.Join(lines, "\n")
}

type Controller struct {
	service *GreetingService
}

func NewController(service *GreetingService) *Controller {
	return &Controller{service: service}
}

func (c *Controller) Hello(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintln(w, c.service.Summary(r.Context()))
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
		Condition(gs.And(
			gs.OnProperty("spring.go-redis.enabled").HavingValue("true").MatchIfMissing(),
			gs.OnProperty("spring.go-redis.addr"),
		)).
		Destroy(CloseRedis).
		Name("__default__")
	gs.Group("${spring.go-redis.instances}", NewRedisClient, CloseRedis)
	gs.Provide(NewGreetingService)
	gs.Provide(NewController)
	gs.Provide(NewHTTPMux)
}

func main() {
	gs.Run()
}
