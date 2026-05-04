package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-spring/spring-core/gs"
)

type fakeRedis struct {
	err   error
	calls int
}

func (f *fakeRedis) Ping(context.Context) error {
	f.calls++
	return f.err
}

func TestGreetingServiceWithFakeRedis(t *testing.T) {
	redis := &fakeRedis{}
	service := &GreetingService{redis: redis, Greeting: "Hi"}

	got := service.Message(context.Background(), "tester")
	if got != "Hi, tester!" {
		t.Fatalf("unexpected greeting: %q", got)
	}
	if redis.calls != 1 {
		t.Fatalf("expected one redis ping, got %d", redis.calls)
	}
}

func TestGreetingServiceDegradedResponse(t *testing.T) {
	redis := &fakeRedis{err: errors.New("redis down")}
	service := &GreetingService{redis: redis, Greeting: "Hi"}

	got := service.Message(context.Background(), "tester")
	if got != "Hi, tester!" {
		t.Fatalf("unexpected greeting: %q", got)
	}
	if redis.calls != 1 {
		t.Fatalf("expected one redis ping, got %d", redis.calls)
	}
}

func TestControllerWithFakeRedis(t *testing.T) {
	service := &GreetingService{redis: &fakeRedis{}, Greeting: "Hi"}
	controller := &Controller{service: service, Audience: "controller"}
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rec := httptest.NewRecorder()

	controller.Hello(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != "Hi, controller!" {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}

func TestIoCContainerWithFakeRedis(t *testing.T) {
	gs.Web(false).Configure(func(app gs.App) {
		app.Property("spring.app.config.dir", "./testdata/empty-conf")
		// The built-in Redis client is not enabled
		app.Provide(&fakeRedis{}).Export(gs.As[RedisPinger]())
	}).RunTest(t, func(ts *struct {
		Service    *GreetingService `autowire:""`
		Controller *Controller      `autowire:""`
	}) {
		if ts.Service == nil {
			t.Fatal("service was not injected")
		}
		if ts.Controller == nil {
			t.Fatal("controller was not injected")
		}
		got := ts.Service.Message(context.Background(), "ioc")
		if got != "Hello, ioc!" {
			t.Fatalf("unexpected ioc greeting: %q", got)
		}
	})
}
