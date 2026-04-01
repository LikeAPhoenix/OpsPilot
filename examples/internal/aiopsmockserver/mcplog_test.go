package aiopsmockserver

import (
	"OpsPilot/examples/internal/aiopsmockdata"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestExecuteQueryLogsMatchesScenarioEvidence(t *testing.T) {
	scenarios, err := aiopsmockdata.LoadScenarios()
	if err != nil {
		t.Fatalf("load scenarios: %v", err)
	}

	now := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)
	output, err := ExecuteQueryLogs("single", scenarios, QueryLogsInput{
		Region:    DefaultRegion,
		TopicID:   DefaultTopicID,
		Query:     "GetBillingDetail response error",
		StartTime: now.Add(-1 * time.Hour),
		EndTime:   now,
		Limit:     defaultLimit,
	}, now)
	if err != nil {
		t.Fatalf("execute query logs: %v", err)
	}
	if !output.Success {
		t.Fatalf("expected success output")
	}
	if output.MatchedCount == 0 {
		t.Fatalf("expected matched logs")
	}
}

func TestExecuteQueryLogsRejectsWrongRegion(t *testing.T) {
	scenarios, err := aiopsmockdata.LoadScenarios()
	if err != nil {
		t.Fatalf("load scenarios: %v", err)
	}

	now := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)
	output, err := ExecuteQueryLogs("multiple", scenarios, QueryLogsInput{
		Region:    "ap-shanghai",
		TopicID:   DefaultTopicID,
		Query:     "panic",
		StartTime: now.Add(-1 * time.Hour),
		EndTime:   now,
		Limit:     defaultLimit,
	}, now)
	if err != nil {
		t.Fatalf("execute query logs: %v", err)
	}
	if output.MatchedCount != 0 {
		t.Fatalf("expected no matched logs, got %d", output.MatchedCount)
	}
}

func TestParseQueryLogsInputRejectsBadRange(t *testing.T) {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"region":     DefaultRegion,
				"topic_id":   DefaultTopicID,
				"query":      "panic",
				"start_time": "2026-03-31T10:00:00Z",
				"end_time":   "2026-03-31T09:00:00Z",
			},
		},
	}

	if _, err := ParseQueryLogsInput(request); err == nil {
		t.Fatalf("expected invalid range error")
	}
}
