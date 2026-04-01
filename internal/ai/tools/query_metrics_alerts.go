package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// PrometheusAlert 对应 Prometheus `/api/v1/alerts` 的原始告警结构。
type PrometheusAlert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	State       string            `json:"state"`
	ActiveAt    string            `json:"activeAt"`
	Value       string            `json:"value"`
}

// PrometheusAlertsResult 表示 Prometheus 告警接口的完整响应。
type PrometheusAlertsResult struct {
	Status string `json:"status"`
	Data   struct {
		Alerts []PrometheusAlert `json:"alerts"`
	} `json:"data"`
	Error     string `json:"error,omitempty"`
	ErrorType string `json:"errorType,omitempty"`
}

// SimplifiedAlert 表示对 LLM 更友好的精简告警信息。
type SimplifiedAlert struct {
	AlertName   string `json:"alert_name" jsonschema:"description=告警名称，从 Prometheus 告警的 labels.alertname 字段提取"`
	Description string `json:"description" jsonschema:"description=告警描述信息，从 Prometheus 告警的 annotations.description 字段提取"`
	State       string `json:"state" jsonschema:"description=告警状态，通常为 'firing'（触发中）或 'pending'（待触发）"`
	ActiveAt    string `json:"active_at" jsonschema:"description=告警激活时间，RFC3339 格式的时间戳，例如 '2025-10-29T08:48:42.496134755Z'"`
	Duration    string `json:"duration" jsonschema:"description=告警持续时间，从激活时间到当前时间的时长，格式如 '2h30m15s'、'30m15s' 或 '15s'"`
}

// PrometheusAlertsOutput 表示工具最终返回给模型的输出结构。
type PrometheusAlertsOutput struct {
	Success bool              `json:"success" jsonschema:"description=查询是否成功"`
	Alerts  []SimplifiedAlert `json:"alerts,omitempty" jsonschema:"description=活动告警列表，每个告警包含名称、描述、状态、激活时间和持续时间。相同 alertname 的告警只保留第一个"`
	Message string            `json:"message,omitempty" jsonschema:"description=操作结果的状态消息"`
	Error   string            `json:"error,omitempty" jsonschema:"description=如果查询失败，包含错误信息"`
}

// NewPrometheusAlertsQueryTool 创建 Prometheus 活跃告警查询工具。
func NewPrometheusAlertsQueryTool(baseURL string, httpClient *http.Client) (tool.BaseTool, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("prometheus base url is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	return utils.InferOptionableTool(
		"query_prometheus_alerts",
		"Query active alerts from Prometheus alerting system. This tool retrieves all currently active/firing alerts including their labels, annotations, state, and values. Use this tool when you need to check what alerts are currently firing, investigate alert conditions, or monitor alert status.",
		func(ctx context.Context, input *struct{}, opts ...tool.Option) (string, error) {
			result, err := queryPrometheusAlerts(ctx, httpClient, baseURL)
			if err != nil {
				output := PrometheusAlertsOutput{
					Success: false,
					Error:   err.Error(),
					Message: "Failed to query Prometheus alerts",
				}
				data, marshalErr := json.MarshalIndent(output, "", "  ")
				if marshalErr != nil {
					return "", err
				}
				return string(data), err
			}

			// 相同 alertname 可能对应多条实例级告警，这里只保留首条，减少模型输入噪音。
			seenAlertNames := make(map[string]bool)
			simplifiedAlerts := make([]SimplifiedAlert, 0, len(result.Data.Alerts))
			for _, alert := range result.Data.Alerts {
				alertName := alert.Labels["alertname"]
				if seenAlertNames[alertName] {
					continue
				}
				seenAlertNames[alertName] = true
				simplifiedAlerts = append(simplifiedAlerts, SimplifiedAlert{
					AlertName:   alertName,
					Description: alert.Annotations["description"],
					State:       alert.State,
					ActiveAt:    alert.ActiveAt,
					Duration:    calculateDuration(alert.ActiveAt),
				})
			}

			output := PrometheusAlertsOutput{
				Success: true,
				Alerts:  simplifiedAlerts,
				Message: fmt.Sprintf("Successfully retrieved %d active alerts", len(simplifiedAlerts)),
			}
			data, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				return "", err
			}
			return string(data), nil
		},
	)
}

// queryPrometheusAlerts 调用 Prometheus HTTP API 拉取原始告警数据。
func queryPrometheusAlerts(ctx context.Context, httpClient *http.Client, baseURL string) (PrometheusAlertsResult, error) {
	var result PrometheusAlertsResult

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/alerts", nil)
	if err != nil {
		return result, err
	}

	resp, err := httpClient.Do(request)
	if err != nil {
		return result, fmt.Errorf("failed to query Prometheus alerts: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return result, fmt.Errorf("prometheus returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return result, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// calculateDuration 将告警激活时间转换为更易读的持续时长。
func calculateDuration(activeAtStr string) string {
	activeAt, err := time.Parse(time.RFC3339Nano, activeAtStr)
	if err != nil {
		return "unknown"
	}

	duration := time.Since(activeAt)
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
