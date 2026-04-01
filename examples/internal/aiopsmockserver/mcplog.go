package aiopsmockserver

import (
	"OpsPilot/examples/internal/aiopsmockdata"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	DefaultRegion  = "ap-guangzhou"
	DefaultTopicID = "869830db-a055-4479-963b-3c898d27e755"
	defaultLimit   = 20
	maxLimit       = 100
)

// QueryLogsInput describes the mock MCP log query input.
type QueryLogsInput struct {
	Region    string
	TopicID   string
	Query     string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
}

type logResult struct {
	Timestamp string            `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// QueryLogsOutput describes the mock MCP log query output.
type QueryLogsOutput struct {
	Success      bool        `json:"success"`
	Message      string      `json:"message"`
	MatchedCount int         `json:"matched_count"`
	Truncated    bool        `json:"truncated"`
	Logs         []logResult `json:"logs"`
}

// NewMCPLogHandler creates the MCP log mock HTTP handler.
func NewMCPLogHandler(listenAddr string, defaultScenario string) (http.Handler, error) {
	scenarios, err := aiopsmockdata.LoadScenarios()
	if err != nil {
		return nil, err
	}
	if err := aiopsmockdata.ValidateScenario(defaultScenario, scenarios); err != nil {
		return nil, err
	}

	mcpServer := server.NewMCPServer(
		"opspilot-mcp-log-mock",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	tool := mcp.NewTool(
		"query_logs",
		mcp.WithDescription("Query mock application logs for local AIOps testing. This tool requires region, topic_id, query, start_time, and end_time. It returns log evidence that matches the selected mock scenario."),
		mcp.WithString("region",
			mcp.Required(),
			mcp.Description("Log region, for example ap-guangzhou."),
		),
		mcp.WithString("topic_id",
			mcp.Required(),
			mcp.Description("Log topic identifier used by the mock service."),
		),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search keywords, for example panic, region mismatch, or GetBillingDetail response error."),
		),
		mcp.WithString("start_time",
			mcp.Required(),
			mcp.Description("Start time in RFC3339 format."),
		),
		mcp.WithString("end_time",
			mcp.Required(),
			mcp.Description("End time in RFC3339 format."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of logs to return. Defaults to 20 and is capped at 100."),
		),
	)

	mcpServer.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input, err := ParseQueryLogsInput(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		scenarioName := defaultScenario
		if queryScenario := request.GetString("scenario", ""); queryScenario != "" {
			scenarioName = queryScenario
		}

		output, err := ExecuteQueryLogs(scenarioName, scenarios, input, time.Now())
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return nil, err
		}

		return mcp.NewToolResultText(string(data)), nil
	})

	return server.NewSSEServer(
		mcpServer,
		server.WithBaseURL(MCPLogPublicBaseURL(listenAddr)),
		server.WithSSEEndpoint("/sse"),
		server.WithMessageEndpoint("/message"),
	), nil
}

// ParseQueryLogsInput parses the query_logs arguments.
func ParseQueryLogsInput(request mcp.CallToolRequest) (QueryLogsInput, error) {
	region, err := request.RequireString("region")
	if err != nil {
		return QueryLogsInput{}, err
	}
	topicID, err := request.RequireString("topic_id")
	if err != nil {
		return QueryLogsInput{}, err
	}
	query, err := request.RequireString("query")
	if err != nil {
		return QueryLogsInput{}, err
	}
	startTimeRaw, err := request.RequireString("start_time")
	if err != nil {
		return QueryLogsInput{}, err
	}
	endTimeRaw, err := request.RequireString("end_time")
	if err != nil {
		return QueryLogsInput{}, err
	}

	startTime, err := time.Parse(time.RFC3339, startTimeRaw)
	if err != nil {
		return QueryLogsInput{}, fmt.Errorf("start_time must be RFC3339: %w", err)
	}
	endTime, err := time.Parse(time.RFC3339, endTimeRaw)
	if err != nil {
		return QueryLogsInput{}, fmt.Errorf("end_time must be RFC3339: %w", err)
	}
	if endTime.Before(startTime) {
		return QueryLogsInput{}, fmt.Errorf("end_time must be after or equal to start_time")
	}

	limit := defaultLimit
	if request.GetArguments()["limit"] != nil {
		limit, err = request.RequireInt("limit")
		if err != nil {
			return QueryLogsInput{}, err
		}
		if limit <= 0 || limit > maxLimit {
			return QueryLogsInput{}, fmt.Errorf("limit must be between 1 and %d", maxLimit)
		}
	}

	return QueryLogsInput{
		Region:    region,
		TopicID:   topicID,
		Query:     query,
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     limit,
	}, nil
}

// ExecuteQueryLogs executes the mock log filtering.
func ExecuteQueryLogs(
	scenarioName string,
	scenarios map[string]aiopsmockdata.Scenario,
	input QueryLogsInput,
	now time.Time,
) (QueryLogsOutput, error) {
	if err := aiopsmockdata.ValidateScenario(scenarioName, scenarios); err != nil {
		return QueryLogsOutput{}, err
	}

	logs, err := aiopsmockdata.BuildLogsForScenario(scenarioName, scenarios, now)
	if err != nil {
		return QueryLogsOutput{}, err
	}

	filtered, matchedCount := filterLogs(logs, input)
	truncated := false
	if len(filtered) > input.Limit {
		filtered = filtered[:input.Limit]
		truncated = true
	}

	resultLogs := make([]logResult, 0, len(filtered))
	for _, entry := range filtered {
		resultLogs = append(resultLogs, logResult{
			Timestamp: entry.Timestamp,
			Level:     entry.Level,
			Message:   entry.Message,
			Metadata:  cloneMetadata(entry.Metadata),
		})
	}

	message := fmt.Sprintf("matched %d logs in scenario %s", matchedCount, scenarioName)
	if matchedCount == 0 {
		message = fmt.Sprintf("no logs matched query in scenario %s", scenarioName)
	}

	return QueryLogsOutput{
		Success:      true,
		Message:      message,
		MatchedCount: matchedCount,
		Truncated:    truncated,
		Logs:         resultLogs,
	}, nil
}

func filterLogs(logs []aiopsmockdata.LogRecord, input QueryLogsInput) ([]aiopsmockdata.LogRecord, int) {
	query := strings.ToLower(input.Query)
	tokens := splitQueryTokens(query)

	filtered := make([]aiopsmockdata.LogRecord, 0, len(logs))
	for _, entry := range logs {
		if entry.Region != input.Region {
			continue
		}
		if entry.TopicID != input.TopicID {
			continue
		}

		timestamp, err := time.Parse(time.RFC3339Nano, entry.Timestamp)
		if err != nil {
			continue
		}
		if timestamp.Before(input.StartTime) || timestamp.After(input.EndTime) {
			continue
		}

		searchable := strings.ToLower(entry.Message + " " + strings.Join(entry.Keywords, " "))
		if !matchesQuery(searchable, query, tokens) {
			continue
		}

		filtered = append(filtered, entry)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp > filtered[j].Timestamp
	})

	return filtered, len(filtered)
}

func matchesQuery(searchable string, rawQuery string, tokens []string) bool {
	if rawQuery == "" {
		return true
	}
	if strings.Contains(searchable, rawQuery) {
		return true
	}
	if len(tokens) == 0 {
		return false
	}
	for _, token := range tokens {
		if !strings.Contains(searchable, token) {
			return false
		}
	}
	return true
}

func splitQueryTokens(query string) []string {
	fields := strings.FieldsFunc(query, func(r rune) bool {
		switch {
		case r >= 'a' && r <= 'z':
			return false
		case r >= '0' && r <= '9':
			return false
		case r >= 'A' && r <= 'Z':
			return false
		}
		return true
	})

	seen := make(map[string]struct{}, len(fields))
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.ToLower(field)
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		result = append(result, field)
	}
	return result
}

func cloneMetadata(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}

	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

// MCPLogPublicBaseURL converts a listen address to a local base URL.
func MCPLogPublicBaseURL(listenAddr string) string {
	if listenAddr[0] == ':' {
		return "http://127.0.0.1" + listenAddr
	}
	return "http://" + listenAddr
}
