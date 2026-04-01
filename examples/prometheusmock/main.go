package main

import (
	"OpsPilot/examples/internal/aiopsmockserver"
	"flag"
	"log"
	"net/http"
)

func main() {
	listenAddr := flag.String("addr", ":19090", "HTTP listen address")
	defaultScenario := flag.String("scenario", "single", "default alert scenario: single, multiple, empty")
	flag.Parse()

	handler, err := aiopsmockserver.NewPrometheusHandler(*defaultScenario)
	if err != nil {
		log.Fatalf("build prometheus mock handler: %v", err)
	}

	log.Printf("Prometheus mock listening on %s", *listenAddr)
	log.Printf("default scenario: %s", *defaultScenario)
	log.Printf("alerts endpoint: %s/api/v1/alerts", aiopsmockserver.PrometheusPublicBaseURL(*listenAddr))

	if err := http.ListenAndServe(*listenAddr, handler); err != nil {
		log.Fatalf("start mock server: %v", err)
	}
}
