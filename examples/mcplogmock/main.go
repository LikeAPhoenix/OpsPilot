package main

import (
	"OpsPilot/examples/internal/aiopsmockserver"
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	listenAddr := flag.String("addr", ":19092", "SSE listen address")
	defaultScenario := flag.String("scenario", "single", "default log scenario: single, multiple, empty")
	flag.Parse()

	handler, err := aiopsmockserver.NewMCPLogHandler(*listenAddr, *defaultScenario)
	if err != nil {
		log.Fatalf("build mcp server: %v", err)
	}

	httpServer := &http.Server{
		Addr:    *listenAddr,
		Handler: handler,
	}

	log.Printf("MCP log mock listening on %s", *listenAddr)
	log.Printf("default scenario: %s", *defaultScenario)
	log.Printf("SSE endpoint: %s/sse", aiopsmockserver.MCPLogPublicBaseURL(*listenAddr))
	log.Printf("Configured test region/topic: %s / %s", aiopsmockserver.DefaultRegion, aiopsmockserver.DefaultTopicID)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(ctx)
	}()

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("start mcp log mock: %v", err)
	}
}
