# Go-Spring 核心用法入门

这套文档和示例面向第一次接触 Go-Spring 的 Go 开发者。示例按阅读顺序放在 `examples/` 下，每个目录都是独立 Go module，并且都通过 `replace` 使用本机源码：

```go
replace github.com/go-spring/spring-core => /Users/didi/Go-Spring/spring-core
```

建议始终进入示例目录运行命令，因为 Go-Spring 默认从当前工作目录的 `./conf/app.*` 加载配置。

如果本地 `spring-core` 的生成 mock 代码在 `go run` 时提示 `Mockey check failed`，可以临时加上 `MOCKEY_CHECK_GCFLAGS=false` 运行；该提示来自本地依赖的运行时检查，不影响本套示例的 Go-Spring 用法。

## 1. 只调用 `gs.Run()`

**本节目标**：认识 Go-Spring 应用的最小启动入口。

**完整示例所在目录**：`examples/01-run-only`

**核心代码说明**：

```go
func main() {
	gs.Run()
}
```

`gs.Run()` 会创建应用并阻塞运行。启动过程中会打印 banner、加载配置、初始化日志、启动 IoC 容器、启动内置 HTTP Server，并监听 `SIGINT` / `SIGTERM` 做优雅关闭。

**运行方式**：

```bash
cd examples/01-run-only
go run .
```

**预期结果**：控制台出现 Go-Spring banner，应用监听默认 HTTP 地址 `:9090`，按 `Ctrl+C` 后退出。

**与上一节相比新增了什么能力**：这是起点，只有应用生命周期，没有业务逻辑。

## 2. 增加标准库 `http.HandleFunc`

**本节目标**：验证 Go-Spring 和 Go 标准库 HTTP 生态兼容。

**完整示例所在目录**：`examples/02-stdlib-http`

**核心代码说明**：

```go
http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("hello from net/http\n"))
})

gs.Run()
```

Go-Spring 内置 HTTP Server 默认会把 `http.DefaultServeMux` 包装成 `*gs.HttpServeMux`，所以标准库的 `http.HandleFunc` 会直接生效。

**运行方式**：

```bash
cd examples/02-stdlib-http
go run .
curl http://127.0.0.1:9090/hello
```

**预期结果**：

```text
hello from net/http
```

**与上一节相比新增了什么能力**：应用可以直接暴露标准库 HTTP 路由。

## 3. 使用 `gs.Configure()` 注册 root bean

**本节目标**：使用 `gs.Configure()` 和 `app.Root()` 把已有对象纳入 IoC 容器，并绑定配置。

**完整示例所在目录**：`examples/03-configure-root-bean`

**核心代码说明**：

```go
type GreetingRoot struct {
	Greeting string `value:"${demo.greeting:=Hello}" expr:"$ != ''"`
	Audience string `value:"${demo.audience:=Go-Spring}" expr:"$ != ''"`
}

root := &GreetingRoot{}
http.HandleFunc("/hello", root.Hello)

gs.Configure(func(app gs.App) {
	app.Root(root)
}).Run()
```

`value` tag 同时展示了配置 key 和默认值，`expr` tag 在启动期校验配置。`app.Root(root)` 会触发 root bean 的配置绑定，HTTP handler 使用同一个 `root` 指针。

**运行方式**：

```bash
cd examples/03-configure-root-bean
go run .
curl http://127.0.0.1:9090/hello
```

**预期结果**：

```text
Hello, Go-Spring!
```

**与上一节相比新增了什么能力**：业务对象开始由 Go-Spring 绑定配置和参与容器生命周期。

## 4. 使用配置文件、环境变量和命令行参数覆盖配置

**本节目标**：理解配置默认值、配置文件、环境变量和命令行参数的优先级。

**完整示例所在目录**：`examples/04-config-overrides`

**核心代码说明**：

```go
type GreetingRoot struct {
	Greeting string `value:"${demo.greeting:=Hello}" expr:"$ != ''"`
	Audience string `value:"${demo.audience:=Go-Spring}" expr:"$ != ''"`
}
```

示例目录包含 `conf/app.properties`：

```properties
demo.greeting=Hello from ./conf/app.properties
demo.audience=config file
```

Go-Spring 默认从 `./conf/app.properties`、`./conf/app.yaml` 等文件加载配置。环境变量使用 `GS_` 前缀，例如 `GS_DEMO_AUDIENCE` 会转换成 `demo.audience`。命令行使用 `-Dkey=value`，优先级最高。

**运行方式**：

```bash
cd examples/04-config-overrides
go run .
GS_DEMO_AUDIENCE="env var" go run .
go run . -Ddemo.audience="cmd arg"
```

**预期结果**：

```text
Hello from ./conf/app.properties, config file!
Hello from ./conf/app.properties, env var!
Hello from ./conf/app.properties, cmd arg!
```

**与上一节相比新增了什么能力**：配置值不再只来自 tag 默认值，而是可以由运行环境覆盖。

## 5. 使用 HTTP 中间件和 `gs.HttpServeMux`

**本节目标**：显式提供 `*gs.HttpServeMux`，接入标准 HTTP 中间件。

**完整示例所在目录**：`examples/05-http-middleware-mux`

**核心代码说明**：

```go
func NewHTTPMux(c *Controller) *gs.HttpServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", c.Hello)
	return &gs.HttpServeMux{Handler: requestID(logging(mux))}
}

func init() {
	gs.Provide(NewController, gs.TagArg("${demo}"))
	gs.Provide(NewHTTPMux)
}
```

当应用需要替换默认 `http.DefaultServeMux`，或者要统一包一层请求日志、耗时统计、请求 ID、panic recovery 等中间件时，就需要显式提供 `*gs.HttpServeMux`。

**运行方式**：

```bash
cd examples/05-http-middleware-mux
go run .
curl -i http://127.0.0.1:9090/hello
```

**预期结果**：响应头包含 `X-Request-ID`，控制台打印请求方法、路径、状态码、耗时和 request ID。

**与上一节相比新增了什么能力**：HTTP 入口可以由容器装配，并统一挂载中间件。

## 6. 多个 bean 的依赖注入

**本节目标**：把单个对象拆成 controller 和 service，并使用构造函数注入。

**完整示例所在目录**：`examples/06-multi-bean-di`

**核心代码说明**：

```go
func NewGreetingService() *GreetingService {
	return &GreetingService{}
}

func NewController(service *GreetingService) *Controller {
	return &Controller{service: service}
}

func init() {
	gs.Provide(NewGreetingService)
	gs.Provide(NewController)
	gs.Provide(NewHTTPMux)
}
```

`Controller` 依赖 `GreetingService`，依赖关系由构造函数声明。controller 仍保留 HTTP 响应所需的配置字段 `demo.audience`。

**运行方式**：

```bash
cd examples/06-multi-bean-di
go run .
curl http://127.0.0.1:9090/hello
```

**预期结果**：

```text
Hello from service, controller config!
```

**与上一节相比新增了什么能力**：对象之间的依赖由 IoC 容器解析，代码更容易测试和替换。

## 7. 使用 `gs.Provide` 注册 Redis 单客户端

**本节目标**：模仿 starter 的写法，用 `gs.Provide` 注册一个默认 Redis 客户端。

**完整示例所在目录**：`examples/07-redis-single-client`

**核心代码说明**：

```go
gs.Provide(NewRedisClient, gs.TagArg("${spring.go-redis}")).
	Condition(gs.OnProperty("spring.go-redis.addr")).
	Destroy(CloseRedis).
	Name("__default__")
```

真实 `starter-go-redis` 使用 go-redis 客户端。为了让示例无需下载额外依赖，这里用标准库实现了一个很小的 Redis PING 客户端，配置结构和注册方式保持接近 starter：从 `spring.go-redis` 绑定配置，命名为 `__default__`，并在配置存在时才启用。

**运行方式**：

```bash
cd examples/07-redis-single-client
go run .
curl http://127.0.0.1:9090/hello
```

**预期结果**：如果本机 `127.0.0.1:6379` 有 Redis，会看到 `redis=PONG`；没有 Redis 时应用仍能启动，响应会包含 `redis unavailable` 和连接错误。

**与上一节相比新增了什么能力**：service 可以注入外部客户端 bean，并在业务逻辑中使用它。

## 8. 条件化启用与多实例客户端

**本节目标**：参考 `starter-go-redis` 的思路，同时演示单例 client、多实例 client 和条件化启用。

**完整示例所在目录**：`examples/08-conditional-multi-redis`

**核心代码说明**：

```go
gs.Provide(NewRedisClient, gs.TagArg("${spring.go-redis}")).
	Condition(gs.And(
		gs.OnProperty("spring.go-redis.enabled").HavingValue("true").MatchIfMissing(),
		gs.OnProperty("spring.go-redis.addr"),
	)).
	Name("__default__")

gs.Group("${spring.go-redis.instances}", NewRedisClient, CloseRedis)
```

`gs.OnProperty("spring.go-redis.addr")` 存在时启用默认客户端；`spring.go-redis.enabled=false` 会关闭默认客户端；`gs.Group("${spring.go-redis.instances}", ...)` 会把配置 map 中的每个条目注册为一个命名 bean，例如 `cache`、`queue`。

**运行方式**：

```bash
cd examples/08-conditional-multi-redis
go run .
curl http://127.0.0.1:9090/hello
go run . -Dspring.go-redis.enabled=false
```

**预期结果**：默认配置会列出 `default`、`cache`、`queue` 三类客户端状态；关闭 `spring.go-redis.enabled` 后默认客户端显示 disabled，但 `instances` 下的多实例客户端仍会注册。

**与上一节相比新增了什么能力**：Bean 是否存在可以由配置控制，同一类客户端也可以按配置批量创建多个实例。

## 9. 日志系统

**本节目标**：使用 Go-Spring 日志系统完成标签注册、标签路由、结构化日志和日志配置。

**完整示例所在目录**：`examples/09-logging`

**核心代码说明**：

```go
var (
	tagBizGreeting = gslog.RegisterBizTag("greeting", "serve")
	tagHTTP        = gslog.RegisterRPCTag("http", "request")
	tagRedis       = gslog.RegisterRPCTag("redis", "ping")
)

gslog.Info(ctx, tagBizGreeting,
	gslog.Int("client_count", len(s.Clients)),
	gslog.Msg("building greeting"),
)
```

示例在启动前通过 `log.RefreshConfig` 加载 logger/appender 配置，把 `_biz_greeting_*`、`_rpc_http_*`、`_rpc_redis_*` 路由到控制台，并使用 `JSONLayout` 输出结构化日志。HTTP 中间件把 request ID 写入 context，日志系统通过 `FieldsFromContext` 自动带上该字段。

**运行方式**：

```bash
cd examples/09-logging
go run .
curl -H 'X-Request-ID: demo-1' http://127.0.0.1:9090/hello
```

**预期结果**：控制台输出 JSON 日志，包含 `tag`、`request_id`、HTTP 字段、Redis ping 字段和业务字段。没有 Redis 时会看到 `_rpc_redis_ping` 的 warn 日志。

**与上一节相比新增了什么能力**：日志从普通文本升级为可按标签路由的结构化事件。

## 10. 单元测试

**本节目标**：展示纯单元测试、IoC 容器测试、断言和 fake。

**完整示例所在目录**：`examples/10-unit-tests`

**核心代码说明**：

```go
type RedisPinger interface {
	Ping(context.Context) error
}

func NewGreetingService(redis RedisPinger) *GreetingService {
	return &GreetingService{redis: redis}
}
```

service 依赖接口，生产环境用 `RedisClient` 实现并通过 `Export(gs.As[RedisPinger]())` 暴露给容器；测试中用 `fakeRedis` 替换。`main_test.go` 包含直接构造 service 的纯单元测试、直接调用 controller 的 HTTP 测试，以及 `gs.Web(false).RunTest` 的 IoC 容器测试。

**运行方式**：

```bash
cd examples/10-unit-tests
go test
```

**预期结果**：

```text
PASS
ok  	example.com/go-spring-first/10-unit-tests
```

**与上一节相比新增了什么能力**：业务组件可以脱离真实 Redis 和真实 HTTP Server 进行测试，也可以在 Go-Spring 容器里验证注入关系。

## 示例验证建议

批量检查所有示例：

```bash
ls examples/*/go.mod
for d in examples/*; do (cd "$d" && go test ./...); done
```

启动类示例会阻塞运行，验证启动时可以加命令行覆盖让 HTTP 监听随机端口：

```bash
cd examples/06-multi-bean-di
go run . -Dspring.http.server.addr=:0
```

Redis 示例不要求本机一定有 Redis；没有 Redis 时仍应能编译和启动，访问接口时返回降级提示。
