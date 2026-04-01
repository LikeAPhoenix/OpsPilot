package main

import (
	"OpsPilot/internal/bootstrap"
	appconfig "OpsPilot/internal/infra/config"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()

	cfg, err := appconfig.Load("")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	cleanup, err := ensureExternalMocks(ctx, cfg)
	if err != nil {
		log.Fatalf("ensure aiops mocks: %v", err)
	}
	defer cleanup()

	app, err := bootstrap.New(ctx, "")
	if err != nil {
		log.Fatalf("bootstrap app: %v", err)
	}
	defer app.Close()
	logger := app.Logger.Named("examples.aiops")

	result, err := app.AIOpsService.Analyze(ctx)
	if err != nil {
		logger.Fatal("aiops analyze failed", zap.Error(err))
	}

	fmt.Println("----- Final Response -----")
	fmt.Println(result.Result)
	fmt.Println("----- Final Detail -----")
	for i, detail := range result.Detail {
		fmt.Printf("[%d] %s\n", i+1, detail)
	}
}

func ensureExternalMocks(ctx context.Context, cfg *appconfig.Config) (func(), error) {
	var started []*exec.Cmd
	cleanup := func() { stopStartedProcesses(started) }

	promReady, err := probePrometheus(cfg.Prometheus.BaseURL)
	if err != nil {
		return cleanup, err
	}
	if promReady {
		log.Printf("reuse existing prometheus mock")
	} else {
		cmd, err := startMockProgram("./examples/prometheusmock")
		if err != nil {
			return cleanup, err
		}
		started = append(started, cmd)
		if err := waitForPrometheus(cfg.Prometheus.BaseURL); err != nil {
			stopStartedProcesses(started)
			return cleanup, err
		}
		log.Printf("start prometheusmock program")
	}

	mcpReady, err := probeMCP(ctx, cfg.MCP.URL)
	if err != nil {
		return cleanup, err
	}
	if mcpReady {
		log.Printf("reuse existing mcp log mock")
	} else {
		cmd, err := startMockProgram("./examples/mcplogmock")
		if err != nil {
			stopStartedProcesses(started)
			return cleanup, err
		}
		started = append(started, cmd)
		if err := waitForMCP(ctx, cfg.MCP.URL); err != nil {
			stopStartedProcesses(started)
			return cleanup, err
		}
		log.Printf("start mcplogmock program")
	}

	return cleanup, nil
}

func startMockProgram(path string) (*exec.Cmd, error) {
	cmd := exec.Command("go", "run", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

func stopStartedProcesses(cmds []*exec.Cmd) {
	for i := len(cmds) - 1; i >= 0; i-- {
		if cmds[i].Process == nil {
			continue
		}
		_ = cmds[i].Process.Kill()
		_, _ = cmds[i].Process.Wait()
	}
}

func probePrometheus(baseURL string) (bool, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(baseURL + "/api/v1/alerts")
	if err != nil {
		return false, nil
	}
	defer response.Body.Close()

	return response.StatusCode == http.StatusOK, nil
}

func waitForPrometheus(baseURL string) error {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		ready, err := probePrometheus(baseURL)
		if err != nil {
			return err
		}
		if ready {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("prometheus mock is unavailable at %s", baseURL)
}

func probeMCP(ctx context.Context, baseURL string) (bool, error) {
	client, err := mcpclient.NewSSEMCPClient(baseURL, mcpclient.WithHTTPClient(&http.Client{Timeout: 2 * time.Second}))
	if err != nil {
		return false, err
	}
	defer client.Close()

	requestCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := client.Start(requestCtx); err != nil {
		return false, nil
	}

	request := mcp.InitializeRequest{}
	request.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	request.Params.ClientInfo = mcp.Implementation{
		Name:    "opspilot-examples-aiops",
		Version: "1.0.0",
	}
	if _, err := client.Initialize(requestCtx, request); err != nil {
		return false, nil
	}

	result, err := client.ListTools(requestCtx, mcp.ListToolsRequest{})
	if err != nil {
		return false, nil
	}

	for _, tool := range result.Tools {
		if tool.Name == "query_logs" {
			return true, nil
		}
	}

	return false, nil
}

func waitForMCP(ctx context.Context, baseURL string) error {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		ready, err := probeMCP(ctx, baseURL)
		if err != nil {
			return err
		}
		if ready {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("mcp log mock is unavailable at %s", baseURL)
}
