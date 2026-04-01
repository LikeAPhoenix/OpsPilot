package chat

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

// Workflow 封装带检索和工具调用能力的聊天工作流。
type Workflow struct {
	chatModel    model.ToolCallingChatModel
	retriever    retriever.Retriever
	tools        []tool.BaseTool
	runner       compose.Runnable[*Input, *schema.Message]
	callback     callbacks.Handler
	systemPrompt string
}

// Config 定义聊天工作流的装配参数。
type Config struct {
	ChatModel    model.ToolCallingChatModel
	Retriever    retriever.Retriever
	Tools        []tool.BaseTool
	Callback     callbacks.Handler
	SystemPrompt string
}

// NewWorkflow 创建聊天工作流，并在启动阶段预编译执行图。
func NewWorkflow(ctx context.Context, config Config) (*Workflow, error) {
	if config.ChatModel == nil {
		return nil, fmt.Errorf("chat model is required")
	}
	if config.Retriever == nil {
		return nil, fmt.Errorf("retriever is required")
	}
	if strings.TrimSpace(config.SystemPrompt) == "" {
		config.SystemPrompt = defaultSystemPrompt
	}

	workflow := &Workflow{
		chatModel:    config.ChatModel,
		retriever:    config.Retriever,
		tools:        append([]tool.BaseTool{}, config.Tools...),
		callback:     config.Callback,
		systemPrompt: config.SystemPrompt,
	}

	runner, err := workflow.buildRunner(ctx)
	if err != nil {
		return nil, fmt.Errorf("compile chat workflow: %w", err)
	}
	workflow.runner = runner

	return workflow, nil
}

// Chat 同步执行聊天工作流并返回完整答案。
func (w *Workflow) Chat(ctx context.Context, query string, history []*schema.Message) (string, error) {
	output, err := w.runner.Invoke(ctx, &Input{
		Query:   query,
		History: history,
	}, w.invokeOptions()...)
	if err != nil {
		return "", err
	}

	return output.Content, nil
}

// Stream 以流式方式执行聊天工作流，并持续回调输出增量内容。
func (w *Workflow) Stream(ctx context.Context, query string, history []*schema.Message, send func(string) error) (string, error) {
	streamReader, err := w.runner.Stream(ctx, &Input{
		Query:   query,
		History: history,
	}, w.invokeOptions()...)
	if err != nil {
		return "", err
	}
	defer streamReader.Close()

	var fullResponse strings.Builder
	for {
		chunk, err := streamReader.Recv()
		if err == io.EOF {
			return fullResponse.String(), nil
		}
		if err != nil {
			return "", err
		}

		// 流式响应结束后需要回写完整答案，因此这里同步累计全部分片内容。
		fullResponse.WriteString(chunk.Content)
		if send != nil {
			if err := send(chunk.Content); err != nil {
				return "", err
			}
		}
	}
}

// buildRunner 组装检索、提示词模板和 ReAct Agent 组成的执行图。
func (w *Workflow) buildRunner(ctx context.Context) (compose.Runnable[*Input, *schema.Message], error) {
	const (
		rewriteModel = "RewriteModel" // 查询重写仅仅用于消息召回
		inputToRAG   = "InputToRag"
		chatTemplate = "ChatTemplate"
		reactAgent   = "ReactAgent"
		retrieverKey = "MilvusRetriever"
		inputToChat  = "InputToChat"
	)

	graph := compose.NewGraph[*Input, *schema.Message]()
	_ = graph.AddLambdaNode(inputToRAG, compose.InvokableLambda(func(ctx context.Context, input *Input) (string, error) {
		// inputToRAG 提取用户查询作为检索输入。
		return input.Query, nil
	}))

	template := prompt.FromMessages(schema.FString,
		schema.SystemMessage(w.systemPrompt),
		schema.MessagesPlaceholder("history", false),
		schema.UserMessage("{content}"),
	)
	_ = graph.AddChatTemplateNode(chatTemplate, template)

	reactLambda, err := w.newReactAgent(ctx)
	if err != nil {
		return nil, err
	}
	_ = graph.AddLambdaNode(reactAgent, reactLambda)
	_ = graph.AddRetrieverNode(retrieverKey, w.retriever, compose.WithOutputKey("documents"))
	_ = graph.AddLambdaNode(inputToChat, compose.InvokableLambda(func(ctx context.Context, input *Input) (map[string]any, error) {
		// inputToChat 组装提示词模板所需的动态变量。
		return map[string]any{
			"content": input.Query,
			"history": input.History,
			"date":    time.Now().Format("2006-01-02 15:04:05"),
		}, nil
	}))

	_ = graph.AddLambdaNode(rewriteModel, compose.InvokableLambda(w.rewriteLambda))

	// 查询既要进入 RAG 检索，也要和历史消息一起参与最终提示词拼装。
	_ = graph.AddEdge(compose.START, inputToChat)
	// 根据是否有历史消息分流到到RAG检索还是需要重写提示词的节点
	_ = graph.AddEdge(compose.START, rewriteModel)
	_ = graph.AddEdge(reactAgent, compose.END)
	_ = graph.AddEdge(rewriteModel, inputToRAG)
	_ = graph.AddEdge(inputToRAG, retrieverKey)
	_ = graph.AddEdge(retrieverKey, chatTemplate)
	_ = graph.AddEdge(inputToChat, chatTemplate)
	_ = graph.AddEdge(chatTemplate, reactAgent)

	return graph.Compile(ctx, compose.WithGraphName("ChatAgent"), compose.WithNodeTriggerMode(compose.AllPredecessor))
}

// newReactAgent 创建负责工具调用和最终回答生成的 ReAct Agent。
func (w *Workflow) newReactAgent(ctx context.Context) (*compose.Lambda, error) {
	config := &react.AgentConfig{
		MaxStep:            25,
		ToolReturnDirectly: map[string]struct{}{},
	}
	config.ToolCallingModel = w.chatModel
	config.ToolsConfig.Tools = append([]tool.BaseTool{}, w.tools...)

	agent, err := react.NewAgent(ctx, config)
	if err != nil {
		return nil, err
	}
	return compose.AnyLambda(agent.Generate, agent.Stream, nil, nil)
}

// 根据是否存在历史信息，决定是否需要重写用户查询以适配知识库检索。
func (w *Workflow) rewriteLambda(ctx context.Context, input *Input) (*Input, error) {
	if len(input.History) == 0 {
		// 没有历史消息时直接使用原始查询进行检索，避免不必要的模型调用和潜在的重写错误。
		return input, nil
	}
	
	messages := []*schema.Message{
		schema.SystemMessage(queryRewritePrompt),
	}
	messages = append(messages, input.History...)
	messages = append(messages, schema.UserMessage("请将当前问题改写为可直接用于知识库检索的查询：\n"+input.Query))

	msg, err := w.chatModel.Generate(ctx, messages)
	if err != nil {
		return nil, err
	}

	return &Input{
		ID:      input.ID,
		Query:   msg.Content,
		History: input.History,
	}, nil
}

// invokeOptions 根据是否配置回调决定是否注入链路日志能力。
func (w *Workflow) invokeOptions() []compose.Option {
	if w.callback == nil {
		return nil
	}
	return []compose.Option{compose.WithCallbacks(w.callback)}
}
