package aiopsplan

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"go.uber.org/zap"
)

// Workflow 封装 Plan-and-Execute 形式的 AIOps 分析工作流。
type Workflow struct {
	plannerModel     model.ToolCallingChatModel
	executorModel    model.ToolCallingChatModel
	tools            []tool.BaseTool
	planExecuteAgent adk.ResumableAgent
	logger           *zap.Logger
}

// Config 定义 AIOps 工作流的装配参数。
type Config struct {
	PlannerModel  model.ToolCallingChatModel
	ExecutorModel model.ToolCallingChatModel
	Tools         []tool.BaseTool
	Logger        *zap.Logger
}

// NewWorkflow 创建 AIOps 分析工作流，并在启动阶段预构建执行 Agent。
func NewWorkflow(ctx context.Context, config Config) (*Workflow, error) {
	if config.PlannerModel == nil {
		return nil, fmt.Errorf("planner model is required")
	}
	if config.ExecutorModel == nil {
		return nil, fmt.Errorf("executor model is required")
	}
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}

	workflow := &Workflow{
		plannerModel:  config.PlannerModel,
		executorModel: config.ExecutorModel,
		tools:         append([]tool.BaseTool{}, config.Tools...),
		logger:        config.Logger.Named("aiops_plan"),
	}

	planExecuteAgent, err := workflow.buildAgent(ctx)
	if err != nil {
		return nil, fmt.Errorf("compile aiops workflow: %w", err)
	}
	workflow.planExecuteAgent = planExecuteAgent

	return workflow, nil
}

// buildAgent 预构建 Plan-and-Execute Agent，避免在请求路径重复创建。
func (w *Workflow) buildAgent(ctx context.Context) (adk.ResumableAgent, error) {
	planner, err := planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: w.plannerModel,
	})
	if err != nil {
		return nil, err
	}

	executor, err := planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: w.executorModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: append([]tool.BaseTool{}, w.tools...),
			},
		},
		MaxIterations: 20,
	})
	if err != nil {
		return nil, err
	}

	replanner, err := planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: w.plannerModel,
	})
	if err != nil {
		return nil, err
	}

	planExecuteAgent, err := planexecute.New(ctx, &planexecute.Config{
		Planner:       planner,
		Executor:      executor,
		Replanner:     replanner,
		MaxIterations: 10,
	})
	if err != nil {
		return nil, fmt.Errorf("build PlanExecuteAgent Error: %w", err)
	}

	return planExecuteAgent, nil
}

// Analyze 运行规划、执行和重规划链路，返回最终结果及过程明细。
func (w *Workflow) Analyze(ctx context.Context, query string) (string, []string, error) {
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: w.planExecuteAgent})
	iter := runner.Query(ctx, query)

	var lastMessage adk.Message
	var detail []string
	var err error
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		w.logger.Debug("planexecute event", zap.Any("event", event))
		if event.Output != nil {
			// 只记录有输出的事件，便于前端或日志查看每一步规划与执行结果。
			lastMessage, _, err = adk.GetMessage(event)
			if err != nil {
				return "", nil, err
			}
			detail = append(detail, lastMessage.String())
		}
	}
	if lastMessage == nil {
		return "", nil, fmt.Errorf("get lastMessage Error")
	}
	return lastMessage.Content, detail, nil
}
