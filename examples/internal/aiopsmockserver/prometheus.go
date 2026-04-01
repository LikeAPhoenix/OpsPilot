package aiopsmockserver

import (
	"OpsPilot/examples/internal/aiopsmockdata"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type apiAlert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	State       string            `json:"state"`
	ActiveAt    string            `json:"activeAt"`
	Value       string            `json:"value"`
}

type alertsResponse struct {
	Status string `json:"status"`
	Data   struct {
		Alerts []apiAlert `json:"alerts"`
	} `json:"data"`
}

// NewPrometheusHandler creates the Prometheus mock HTTP handler.
func NewPrometheusHandler(defaultScenario string) (http.Handler, error) {
	scenarios, err := aiopsmockdata.LoadScenarios()
	if err != nil {
		return nil, err
	}
	if err := aiopsmockdata.ValidateScenario(defaultScenario, scenarios); err != nil {
		return nil, err
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = fmt.Fprintf(
			w,
			"Prometheus mock server\n\nGET /api/v1/alerts\nGET /-/healthy\n\nDefault scenario: %s\nAvailable scenarios: %s\nOverride scenario with query string: /api/v1/alerts?scenario=multiple\n",
			defaultScenario,
			strings.Join(aiopsmockdata.AvailableScenarios(scenarios), ", "),
		)
	})

	mux.HandleFunc("/-/healthy", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/api/v1/alerts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		scenarioName := r.URL.Query().Get("scenario")
		if scenarioName == "" {
			scenarioName = defaultScenario
		}

		alerts, err := buildAlertsForScenario(scenarioName, scenarios, time.Now())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		response := alertsResponse{Status: "success"}
		response.Data.Alerts = alerts

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	return loggingMiddleware(mux), nil
}

func buildAlertsForScenario(name string, scenarios map[string]aiopsmockdata.Scenario, now time.Time) ([]apiAlert, error) {
	built, err := aiopsmockdata.BuildAlertsForScenario(name, scenarios, now)
	if err != nil {
		return nil, err
	}

	alerts := make([]apiAlert, 0, len(built))
	for _, item := range built {
		alerts = append(alerts, apiAlert{
			Labels:      cloneStringMap(item.Labels),
			Annotations: cloneStringMap(item.Annotations),
			State:       item.State,
			ActiveAt:    item.ActiveAt,
			Value:       item.Value,
		})
	}

	return alerts, nil
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}

	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.String(), time.Since(start).Round(time.Millisecond))
	})
}

// PrometheusPublicBaseURL converts a listen address to a local base URL.
func PrometheusPublicBaseURL(listenAddr string) string {
	if listenAddr[0] == ':' {
		return "http://127.0.0.1" + listenAddr
	}
	return "http://" + listenAddr
}
