package main

import "github.com/go-spring/spring-core/gs"

func main() {
	// gs.Run starts a Go-Spring application with the default setup.
	// It prints the startup banner, loads and refreshes application properties,
	// initializes logging, refreshes the IoC container, wires registered beans,
	// runs any Runner beans, starts configured servers, waits for SIGINT/SIGTERM,
	// and then performs a graceful shutdown by stopping servers, closing the
	// container, and destroying the logging system.
	gs.Run()
}

// Start it with `go run main.go`.
// Then try `curl http://127.0.0.1:9090/`.
// It should return `404 page not found`.
// That means the app is running, but no route has been registered yet.
// Press `Ctrl+C` when you want to stop it.
