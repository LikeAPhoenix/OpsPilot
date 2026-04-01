package aiopsmockdata

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

//go:embed testdata/scenarios.json
var embeddedScenarios []byte

// Scenario 描述一组联调用的告警与日志证据。
type Scenario struct {
	Alerts []AlertTemplate `json:"alerts"`
	Logs   []LogTemplate   `json:"logs"`
}

// AlertTemplate 定义告警模板，运行时会基于 activeAgo 生成 activeAt。
type AlertTemplate struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	State       string            `json:"state"`
	ActiveAgo   string            `json:"activeAgo"`
	Value       string            `json:"value"`
}

// LogTemplate 定义日志模板，运行时会基于 timestampAgo 生成 timestamp。
type LogTemplate struct {
	TimestampAgo string            `json:"timestampAgo"`
	Level        string            `json:"level"`
	Message      string            `json:"message"`
	Region       string            `json:"region"`
	TopicID      string            `json:"topic_id"`
	Keywords     []string          `json:"keywords"`
	Metadata     map[string]string `json:"metadata"`
}

// Alert 表示运行时输出的告警数据。
type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	State       string            `json:"state"`
	ActiveAt    string            `json:"activeAt"`
	Value       string            `json:"value"`
}

// LogRecord 表示运行时输出的日志数据。
type LogRecord struct {
	Timestamp string            `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Region    string            `json:"region"`
	TopicID   string            `json:"topic_id"`
	Keywords  []string          `json:"keywords"`
	Metadata  map[string]string `json:"metadata"`
}

// LoadScenarios 读取并校验内置联调场景。
func LoadScenarios() (map[string]Scenario, error) {
	var scenarios map[string]Scenario
	if err := json.Unmarshal(embeddedScenarios, &scenarios); err != nil {
		return nil, fmt.Errorf("unmarshal embedded scenarios: %w", err)
	}
	if err := validateScenarios(scenarios); err != nil {
		return nil, err
	}
	return scenarios, nil
}

// AvailableScenarios 返回排序后的场景名。
func AvailableScenarios(scenarios map[string]Scenario) []string {
	names := make([]string, 0, len(scenarios))
	for name := range scenarios {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ValidateScenario 校验场景名是否存在。
func ValidateScenario(name string, scenarios map[string]Scenario) error {
	if _, ok := scenarios[name]; ok {
		return nil
	}
	return fmt.Errorf("unknown scenario %q, available: %s", name, strings.Join(AvailableScenarios(scenarios), ", "))
}

// BuildAlertsForScenario 生成 Prometheus mock 使用的运行时告警数据。
func BuildAlertsForScenario(name string, scenarios map[string]Scenario, now time.Time) ([]Alert, error) {
	scenario, ok := scenarios[name]
	if !ok {
		return nil, ValidateScenario(name, scenarios)
	}

	alerts := make([]Alert, 0, len(scenario.Alerts))
	for _, item := range scenario.Alerts {
		activeAgo := strings.TrimSpace(item.ActiveAgo)
		if activeAgo == "" {
			activeAgo = "5m"
		}
		duration, err := time.ParseDuration(activeAgo)
		if err != nil {
			return nil, fmt.Errorf("scenario %q has invalid activeAgo %q: %w", name, item.ActiveAgo, err)
		}

		state := strings.TrimSpace(item.State)
		if state == "" {
			state = "firing"
		}

		value := strings.TrimSpace(item.Value)
		if value == "" {
			value = "1e+00"
		}

		alerts = append(alerts, Alert{
			Labels:      cloneStringMap(item.Labels),
			Annotations: cloneStringMap(item.Annotations),
			State:       state,
			ActiveAt:    now.Add(-duration).UTC().Format(time.RFC3339Nano),
			Value:       value,
		})
	}

	return alerts, nil
}

// BuildLogsForScenario 生成日志 mock 使用的运行时日志数据。
func BuildLogsForScenario(name string, scenarios map[string]Scenario, now time.Time) ([]LogRecord, error) {
	scenario, ok := scenarios[name]
	if !ok {
		return nil, ValidateScenario(name, scenarios)
	}

	logs := make([]LogRecord, 0, len(scenario.Logs))
	for _, item := range scenario.Logs {
		timestampAgo := strings.TrimSpace(item.TimestampAgo)
		if timestampAgo == "" {
			timestampAgo = "5m"
		}
		duration, err := time.ParseDuration(timestampAgo)
		if err != nil {
			return nil, fmt.Errorf("scenario %q has invalid timestampAgo %q: %w", name, item.TimestampAgo, err)
		}

		logs = append(logs, LogRecord{
			Timestamp: now.Add(-duration).UTC().Format(time.RFC3339Nano),
			Level:     strings.TrimSpace(item.Level),
			Message:   strings.TrimSpace(item.Message),
			Region:    strings.TrimSpace(item.Region),
			TopicID:   strings.TrimSpace(item.TopicID),
			Keywords:  append([]string{}, item.Keywords...),
			Metadata:  cloneStringMap(item.Metadata),
		})
	}

	return logs, nil
}

func validateScenarios(scenarios map[string]Scenario) error {
	if len(scenarios) == 0 {
		return fmt.Errorf("no scenarios configured")
	}

	for name, scenario := range scenarios {
		for index, alert := range scenario.Alerts {
			if strings.TrimSpace(alert.Labels["alertname"]) == "" {
				return fmt.Errorf("scenario %q alert %d missing labels.alertname", name, index)
			}
		}
		for index, entry := range scenario.Logs {
			if strings.TrimSpace(entry.Message) == "" {
				return fmt.Errorf("scenario %q log %d missing message", name, index)
			}
			if strings.TrimSpace(entry.Region) == "" {
				return fmt.Errorf("scenario %q log %d missing region", name, index)
			}
			if strings.TrimSpace(entry.TopicID) == "" {
				return fmt.Errorf("scenario %q log %d missing topic_id", name, index)
			}
		}
	}

	return nil
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
