# Getting Started with Core Go-Spring Usage

[English](README.en.md) | [中文](README.md)

This is a step-by-step guide. We start with an empty application that can only boot, and each step adds one capability:
first make HTTP routing work, then hand objects over to the container,
then add configuration, dependency injection, external clients, conditional registration, structured logging, and finally tests.
Each step can be run independently, and the complete code is available in the [examples](examples) directory.

## 1. Start a Minimal Go-Spring Application

In the first step, we will not write any business code yet. We only verify how to start a Go-Spring application.

The code is:

```go
func main() {
	gs.Run()
}
```

> Complete code: [examples/01-run-only/main.go](examples/01-run-only/main.go).

Although the code above looks very short, it is already enough for the program to enter the Go-Spring application lifecycle.
`gs.Run()` creates the application, loads configuration, initializes logging, refreshes the IoC container, starts the built-in HTTP server,
listens for `SIGINT` / `SIGTERM`, and finally performs graceful shutdown when the process exits.

Run the example with:

```bash
cd examples/01-run-only
go run .
```

At this point, the console prints output like this:

```text
   ____    ___            ____    ____    ____    ___   _   _    ____ 
  / ___|  / _ \          / ___|  |  _ \  |  _ \  |_ _| | \ | |  / ___|
 | |  _  | | | |  _____  \___ \  | |_) | | |_) |  | |  |  \| | | |  _ 
 | |_| | | |_| | |_____|  ___) | |  __/  |  _ <   | |  | |\  | | |_| |
  \____|  \___/          |____/  |_|     |_| \_\ |___| |_| \_|  \____| 

            go-spring@v1.3.0  https://github.com/go-spring/

[INFO][2026-05-02T19:13:07.837][...ing/spring-core/gs/internal/gs_app/app.go:289] _app_def||msg=ready to serve requests
```

`ready to serve requests` means the application has started and is listening on `:9090`.
Use the following command to access the root path:

```bash
curl http://127.0.0.1:9090/
```

You will get:

```text
404 page not found
```

The 404 here is expected. It shows that the HTTP server has started, but no handler can process this path yet.
Press `Ctrl+C` to stop the program, after which Go-Spring enters its shutdown flow.

## 2. Add a Standard Library HTTP Route

The application from the previous chapter can already start, but it has no business entry point yet, so every request returns 404.
Now, before introducing IoC, we register a plain HTTP handler using the Go standard library.

The code is:

```go
func main() {
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello from net/http\n"))
	})

	gs.Run()
}
```

> Complete code: [examples/02-stdlib-http/main.go](examples/02-stdlib-http/main.go).

Run the example with:

```bash
cd examples/02-stdlib-http
go run .
```

Then access the new `/hello` route:

```bash
curl http://127.0.0.1:9090/hello
```

This time it is no longer a 404. Instead, it returns the expected response:

```text
hello from net/http
```

The application has now moved from "can only start" to "can handle HTTP requests".
However, the handler is still an anonymous function, so there is nowhere clean to put business state or configuration.

## 3. Register a Business Object as a Root Bean

In the previous chapter, `/hello` was an anonymous function written directly inside `main`.
It proves that HTTP handling works, but it is not easy to extend.
Once the greeting text, target audience, or validation rules need to become configurable, the anonymous function starts to feel awkward.
So this chapter creates a business object named `GreetingRoot`, lets it hold configuration, and implements its method as the handler.

The code is:

```go
type GreetingRoot struct {
	Greeting string `value:"${demo.greeting:=Hello}" expr:"$ != ''"`
	Audience string `value:"${demo.audience:=Go-Spring}" expr:"$ != ''"`
}

func (g *GreetingRoot) Hello(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintf(w, "%s, %s!\n", g.Greeting, g.Audience)
}

func main() {
	root := &GreetingRoot{}
	http.HandleFunc("/hello", root.Hello)

	gs.Configure(func(app gs.App) {
		app.Root(root)
	}).Run()
}
```

> Complete code: [examples/03-configure-root-bean/main.go](examples/03-configure-root-bean/main.go).

Compared with the previous chapter, this code has two substantial changes:
- The handler is no longer an anonymous function. It is the `GreetingRoot.Hello` method, and business state enters the struct.
- `root` is passed to `app.Root(root)`, so Go-Spring processes its field tags during startup.

The `value` tag on the `GreetingRoot` fields defines configuration binding:
- `${demo.greeting:=Hello}` means read the value of configuration key `demo.greeting`; if it is not configured anywhere, use the default value `Hello`.
- `expr:"$ != ''"` means the bound value cannot be empty. If the condition is not satisfied, the application fails during startup instead of exposing the problem only after a request arrives.

Run the example with:

```bash
cd examples/03-configure-root-bean
go run .
```

Then access the `/hello` route:

```bash
curl http://127.0.0.1:9090/hello
```

You will get the expected response:

```text
Hello, Go-Spring!
```

Both `Hello` and `Go-Spring` come from the default values in the field tags.
In other words, although the application still uses standard library routing, the business object has already entered Go-Spring's configuration binding flow.

## 4. Override Default Values with External Configuration

In the previous chapter, we wrote the configuration binding relationship into `GreetingRoot`, but the runtime result still relied entirely on the default values in the tags.
Real applications usually do not run only on default values. Differences between environments are often placed in configuration files, environment variables, or startup arguments.
This chapter keeps using the code from the previous chapter without any code changes. It only adds a configuration file to the example directory.

`GreetingRoot` still binds the same two configuration keys:

```go
type GreetingRoot struct {
	Greeting string `value:"${demo.greeting:=Hello}" expr:"$ != ''"`
	Audience string `value:"${demo.audience:=Go-Spring}" expr:"$ != ''"`
}
```

A new configuration file, [app.properties](examples/04-config-overrides/conf/app.properties), is added under the `./conf` directory:

```properties
demo.greeting=Hello from ./conf/app.properties
demo.audience=config file
```

> Complete code: [examples/04-config-overrides/main.go](examples/04-config-overrides/main.go).

Run the example with:

```bash
cd examples/04-config-overrides
go run .
```

Then access the `/hello` route:

```bash
curl http://127.0.0.1:9090/hello
```

The response now changes from the default values to the values in the configuration file:

```text
Hello from ./conf/app.properties, config file!
```

Without changing the configuration file, we can override one configuration key with an environment variable,
for example `GS_DEMO_AUDIENCE`, which maps to `demo.audience`:

```bash
GS_DEMO_AUDIENCE="env var" go run .
curl http://127.0.0.1:9090/hello
```

After running the commands above, the response changes from the value in the configuration file to the value from the environment variable:

```text
Hello from ./conf/app.properties, env var!
```

We can also override configuration with command-line arguments, using the `-Dkey=value` form:

```bash
go run . -Ddemo.audience="cmd arg"
curl http://127.0.0.1:9090/hello
```

After running the commands above, the response changes from the value in the configuration file to the value from the command-line argument:

```text
Hello from ./conf/app.properties, cmd arg!
```

In this chapter, without changing any code, we made the configuration binding from the previous chapter much easier to operate.
However, although `GreetingRoot` can now have configuration bound by Go-Spring, it is still created manually in `main`.

## 5. Assemble the HTTP Mux and Middleware with the Container

In the previous chapters, we kept writing object creation and route registration inside the `main` function.
That is suitable for getting started, but once cross-cutting logic such as request logging, latency metrics, request IDs, and panic recovery appears,
the HTTP entry point should no longer be scattered inside `main`.
So this chapter uses Go-Spring to construct all components.

First, we collect configuration into a configuration struct.
Notice that the tags here no longer use `demo.greeting`; they use `greeting` and `audience`,
because the overall prefix `${demo}` is specified when the constructor is registered.

The code is:

```go
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
```

Then we explicitly create a `*gs.HttpServeMux`.
Internally it still uses the standard library `http.NewServeMux()`, but the handler finally returned to Go-Spring includes middleware.

The code is:

```go
func NewHTTPMux(c *Controller) *gs.HttpServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", c.Hello)
	return &gs.HttpServeMux{Handler: logging(mux)}
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("method=%s path=%s elapsed=%s",
			r.Method, r.URL.Path, time.Since(start))
	})
}
```

This time, we added a `logging` middleware that records the request method, path, and elapsed time.

Finally, we register the constructors with the container:

```go
func init() {
	gs.Provide(NewController, gs.TagArg("${demo}"))
	gs.Provide(NewHTTPMux)
}

func main() {
	gs.Run()
}
```

> Complete code: [examples/05-http-middleware-mux/main.go](examples/05-http-middleware-mux/main.go).

Run the example with:

```bash
cd examples/05-http-middleware-mux
go run .
```

Then access the `/hello` route:

```bash
curl -i http://127.0.0.1:9090/hello
```

The response now looks like this:

```text
Hello with middleware, custom mux!
```

At the same time, the console prints the request method, path, and elapsed time, showing that the request did pass through the `logging` middleware.

This chapter completes an important turn:
`main` goes back to only being responsible for `gs.Run()`, while object creation, configuration binding, and HTTP mux assembly are all handed over to the container.

## 6. Split the Controller and Service into Multiple Beans

In the previous chapter, the container already created the controller and HTTP mux, but the greeting was still assembled by the controller itself.
As the business grows, the controller should focus more on HTTP requests and responses, while business logic and rules should live in a service.
So this chapter adds a `GreetingService`, and the controller depends on the service through constructor injection.

The code is:

```go
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
```

The registration code is also simple. We only need to provide one more constructor:

```go
func init() {
	gs.Provide(NewGreetingService)
	gs.Provide(NewController)
	gs.Provide(NewHTTPMux)
}
```

> Complete code: [examples/06-multi-bean-di/main.go](examples/06-multi-bean-di/main.go).

Run the example with:

```bash
cd examples/06-multi-bean-di
go run .
```

Then access the `/hello` route:

```bash
curl http://127.0.0.1:9090/hello
```

You can see the expected response:

```text
Hello from service, controller config!
```

This chapter demonstrates constructor injection.
The parameters of `NewController` declare that it needs `*GreetingService`,
so Go-Spring creates the service first and then passes it to the controller.
Business code does not need to look up dependencies by itself or manually assemble the object graph in `main`.

## 7. Register an External Client Bean

The service in the previous chapter only had one field, but real services usually depend on external clients such as Redis, databases, and message queues.
To keep the example focused on Go-Spring's registration approach, this chapter uses a lightweight `RedisClient` to simulate an external client:
it reads configuration and prints logs, but does not connect to a real Redis instance.

First, define Redis configuration and the client constructor:

```go
type RedisConfig struct {
	Addr     string `value:"${addr}" expr:"$ != ''"`
	Password string `value:"${password:=}"`
}

type RedisClient struct {
	cfg RedisConfig
}

func NewRedisClient(cfg RedisConfig) (*RedisClient, error) {
	log.Printf("create redis client addr=%s", cfg.Addr)
	return &RedisClient{cfg: cfg}, nil
}

func CloseRedis(*RedisClient) error {
	return nil
}

func (c *RedisClient) Ping(context.Context) error {
	log.Printf("redis ping addr=%s", c.cfg.Addr)
	return nil
}
```

Then make the service depend on `*RedisClient` and call it while handling requests:

```go
type GreetingService struct {
	redis    *RedisClient
	Greeting string `value:"${demo.greeting:=Hello}" expr:"$ != ''"`
}

func NewGreetingService(redis *RedisClient) *GreetingService {
	return &GreetingService{redis: redis}
}

func (s *GreetingService) Message(ctx context.Context, audience string) string {
	_ = s.redis.Ping(ctx)
	return fmt.Sprintf("%s, %s!", s.Greeting, audience)
}
```

Finally, add the Redis client registration code:

```go
func init() {
	gs.Provide(NewRedisClient, gs.TagArg("${spring.go-redis}")).Destroy(CloseRedis)
	gs.Provide(NewGreetingService)
	gs.Provide(NewController)
	gs.Provide(NewHTTPMux)
}
```

When registering the Redis client:
- `gs.TagArg("${spring.go-redis}")` means the constructor parameter `RedisConfig` reads configuration from the `spring.go-redis` prefix.
- `Destroy(CloseRedis)` means the container calls the destroy function when it shuts down.

> Complete code: [examples/07-redis-single-client/main.go](examples/07-redis-single-client/main.go).

We also need to add one configuration key to the configuration file to specify the Redis address:

```properties
spring.go-redis.addr=127.0.0.1:6379
```

Run the example with:

```bash
cd examples/07-redis-single-client
go run .
```

The console prints a log showing that the client was created:

```text
create redis client addr=127.0.0.1:6379
```

Access the `/hello` route:

```bash
curl http://127.0.0.1:9090/hello
```

You can see the expected response:

```text
Hello with Redis, single client!
```

You can also see the `redis ping` log printed during request handling in the console.
Although the Redis client in this chapter is only a simulated object, its registration approach is the same as a real client:
configuration binding, dependency injection, and resource destruction are all handled by the container.

## 8. Conditional Registration and Multi-Instance Clients

The previous chapter registered only one Redis client, so it was enough for the service to depend directly on `*RedisClient`.
But in real applications, it is more common to have multiple instances of the same client type, such as default Redis, cache Redis, and queue Redis.
If we keep writing multiple `NewRedisClient` registrations manually, registration quickly becomes messy. So this chapter introduces conditional registration, named beans, and configuration grouping.

First, add two declarations when registering the default client:

```go
gs.Provide(NewRedisClient, gs.TagArg("${spring.go-redis}")).
	Condition(gs.OnProperty("spring.go-redis.addr")).
	Destroy(CloseRedis).
	Name("__default__")
```

- `Condition(gs.OnProperty("spring.go-redis.addr"))` means the default client is created only when `spring.go-redis.addr` exists in the configuration.
- `Name("__default__")` gives this bean a name, so when there are more instances of the same type later, the injection side can choose it explicitly.

Then register other Redis instances. Instead of writing each registration manually, we hand this to `gs.Group`,
which can create multiple instances of the same type in batches according to configuration:

```go
gs.Group("${spring.go-redis.instances}", NewRedisClient, CloseRedis)
```

We need to add a `spring.go-redis.instances` configuration item to the configuration file. It is a map:
the key is the instance name, and the value is that instance's configuration.

```properties
spring.go-redis.addr=127.0.0.1:6379
spring.go-redis.instances.cache.addr=127.0.0.1:6380
spring.go-redis.instances.queue.addr=127.0.0.1:6381
```

Now the service needs a small adjustment, because there are now multiple registered instances of the same type `*RedisClient`.
We can use the `autowire` tag on its field to specify that the instance named `__default__` should be injected:

```go
type GreetingService struct {
	Client   *RedisClient `autowire:"__default__?"`
	Greeting string       `value:"${demo.greeting:=Hello}" expr:"$ != ''"`
}
```

> Complete code: [examples/08-conditional-multi-redis/main.go](examples/08-conditional-multi-redis/main.go).

Run the example with:

```bash
cd examples/08-conditional-multi-redis
go run .
```

The console prints the log for creating `__default__`, but it does not print creation logs for `cache` or `queue`.
This is because Go-Spring instantiates beans on demand. Instances that are not used are not created.

Access the `/hello` route:

```bash
curl http://127.0.0.1:9090/hello
```

You can see the expected response:

```text
Hello with conditional Redis, conditional clients!
```

We can modify the service to inject the `cache` instance:

```go
type GreetingService struct {
	Client   *RedisClient `autowire:"cache?"`
	Greeting string       `value:"${demo.greeting:=Hello}" expr:"$ != ''"`
}
```

Then only the `cache` instance is created during startup.
In the same way, we can also inject the `queue` instance.

This chapter solves the problem of how to manage multiple instances of the same type.
`Condition` controls whether a bean is created,
`Name` gives a bean a name,
`autowire` lets the dependent side choose a specific instance,
and `Group` converts a group of configuration items into a group of clients in batches.

## 9. Integrate Structured Logging

So far, the examples have demonstrated HTTP, configuration, dependency injection, and client registration, but logging is still plain text.
Real services need logs that are easier to search and correlate: business logs should identify their source, request logs should record method, path, and elapsed time,
and logs from the same request should preferably carry the same request ID.
So this chapter introduces Go-Spring's logging system.

First, register two log tags: one for business logs and one for HTTP access logs:

```go
var (
	tagBizGreeting = log.RegisterBizTag("greeting", "serve")
	tagHTTPRequest = log.RegisterRPCTag("http", "request")
)
```

The service no longer prints logs with the standard library. Instead, it records structured fields with Go-Spring's logging system:

```go
func (s *GreetingService) Summary(ctx context.Context) string {
	log.Info(ctx, tagBizGreeting,
		log.String("greeting", s.Greeting),
		log.Msg("building greeting"),
	)
	return s.Greeting + ", structured logs!"
}
```

Add a `requestID` middleware for the HTTP entry point. It reads or generates a request ID from the request headers.
For information such as request IDs, we want it to be recorded in logs automatically instead of manually adding it every time we print a log.
So we put the request ID into the context, making it convenient for the logging system to extract automatically.

```go
type requestIDKey struct{}

func NewHTTPMux(c *Controller) *gs.HttpServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", c.Hello)
	return &gs.HttpServeMux{Handler: requestID(logging(mux))}
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
```

We need to set a context extraction callback, `log.FieldsFromContext`, for the logging system.
This lets it automatically extract the request ID from the context and record it together with other fields.

```go
log.FieldsFromContext = func(ctx context.Context) []log.Field {
	id, ok := ctx.Value(requestIDKey{}).(string)
	if !ok || id == "" {
		return nil
	}
	return []log.Field{log.String("request_id", id)}
}
```

> Complete code: [examples/09-logging/main.go](examples/09-logging/main.go).

Finally, add logging system configuration to the configuration file so logs are output to the console in JSON format.

```properties
logging.logger.root.type=ConsoleLogger
logging.logger.root.level=INFO
logging.logger.root.layout.type=JSONLayout
logging.logger.root.layout.fileLineMaxLength=30
```

Run the example with:

```bash
cd examples/09-logging
go run .
```

Access the `/hello` route with a request ID:

```bash
curl -H "X-Request-ID: demo-1" http://127.0.0.1:9090/hello
```

You can see the expected response:

```text
Hello with logging, structured logs!
```

At the same time, the console outputs JSON logs containing the tag, request_id, HTTP method, path, elapsed time, and business fields.

```text
{"level":"info","time":"2026-05-03T08:57:21.525","fileLine":"...mples/09-logging/main.go:29","tag":"_biz_greeting_serve","request_id":"demo-1","greeting":"Hello with logging","msg":"building greeting"}
{"level":"info","time":"2026-05-03T08:57:21.526","fileLine":"...mples/09-logging/main.go:80","tag":"_rpc_http_request","request_id":"demo-1","method":"GET","path":"/hello","elapsed":"783.125us","msg":"http request completed"}
```

The point of this chapter is not to "print more content", but to make logs structured events:
tags describe event types, fields carry searchable data, and context connects common fields within the same request.

## 10. Make Components Testable Without Real Services

After the previous steps, the application already has the core structure common to a web service:
HTTP entry point, controller/service layering, configuration binding, external clients, and structured logging.
One final problem remains: **testing**.

If the service depends directly on a concrete Redis client, it is hard to replace during tests.
If tests must start a real HTTP server, feedback also becomes slower.
This chapter changes the dependency into an interface and uses Go-Spring's test container to verify assembly relationships.

The first change is to define an interface, so the service depends on behavior instead of a concrete implementation:

```go
type RedisPinger interface {
	Ping(context.Context) error
}

type GreetingService struct {
	redis    RedisPinger
	Greeting string `value:"${demo.greeting:=Hello}" expr:"$ != ''"`
}

func NewGreetingService(redis RedisPinger) *GreetingService {
	return &GreetingService{redis: redis}
}
```

In production, we still use `RedisClient`, but this time it also needs to be exported as `RedisPinger` during registration:

```go
gs.Provide(NewRedisClient, gs.TagArg("${spring.go-redis}")).
	Condition(gs.OnProperty("spring.go-redis.addr")).
	Destroy(CloseRedis).
	Export(gs.As[RedisPinger]())
```

This lets us use a small fakeRedis in test code to replace the real Redis:

```go
type fakeRedis struct {
	err   error
	calls int
}

func (f *fakeRedis) Ping(context.Context) error {
	f.calls++
	return f.err
}
```

With this fakeRedis, the service can be tested directly:

```go
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
```

For the controller, we can also test the handler with `httptest` without starting a real HTTP server:

```go
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
```

The tests above are pure native Go unit tests and do not depend on the Go-Spring container.
If we also want to verify assembly relationships inside the Go-Spring container, we can follow these steps:

- First use `gs.Web(false)` to disable the real HTTP server.
- Then use `app.Provide(&fakeRedis{}).Export(...)` to register fakeRedis as the interface implementation.
- Finally, inject the objects to inspect inside `gs.RunTest()`.

The code is:

```go
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
```

When `gs.RunTest()` runs, it starts the complete Go-Spring container and assembles the objects registered in `init`.
It accepts a callback function whose parameter is a struct used to inject the objects to inspect.
You can use `autowire` and `value` tags to inject objects or configuration.

> Complete code: [examples/10-unit-tests/main.go](examples/10-unit-tests/main.go),  
> test code: [examples/10-unit-tests/main_test.go](examples/10-unit-tests/main_test.go).

Run the tests with:

```bash
cd examples/10-unit-tests
go test
```

All tests should pass.

This chapter connects all previous capabilities back to testability.
Interfaces let external dependencies be replaced by fakeRedis, `Export(gs.As[...])` lets the production implementation enter the container through an interface,
and `gs.Web(false)` plus `gs.RunTest()` make the container assembly itself testable.

At this point, the complete path of a Go-Spring application has been connected:
minimal startup, HTTP routing, configuration binding, container assembly, external clients, conditional multi-instance registration,
structured logging, and testing.

## Summary

From the ten examples above, we can see that Go-Spring's core value is not replacing the standard libraries and tools that already exist in the Go ecosystem,
but providing a highly engineering-oriented way to organize them:
application startup, configuration binding, object assembly, resource lifecycle, logging, and testing are all organized together.

For very small programs, directly using the standard library may already be enough.
But as service scale continues to grow, these container and lifecycle capabilities gradually show their value.
