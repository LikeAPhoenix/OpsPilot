package sessionsummary

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"OpsPilot/internal/domain/session"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

const defaultSystemPrompt = `
你是会话记忆压缩助手。你的任务是把已有摘要和若干轮更早的对话压缩成一段新的短期记忆摘要。

要求：
1. 只保留与后续连续对话相关的事实、约束、偏好、待办、已确认结论和未解决问题。
2. 不要编造原对话中不存在的信息。
3. 输出纯文本，不要使用 markdown。
4. 控制在 300 字以内，内容紧凑。
`

// Workflow 封装会话摘要生成能力。
type Workflow struct {
	chatModel    model.BaseChatModel
	callback     callbacks.Handler
	systemPrompt string
}

// Config 定义摘要工作流装配参数。
type Config struct {
	ChatModel    model.BaseChatModel
	Callback     callbacks.Handler
	SystemPrompt string
}

// NewWorkflow 创建会话摘要工作流。
func NewWorkflow(ctx context.Context, config Config) (*Workflow, error) {
	_ = ctx
	if config.ChatModel == nil {
		return nil, errors.New("summary chat model is required")
	}
	if strings.TrimSpace(config.SystemPrompt) == "" {
		config.SystemPrompt = defaultSystemPrompt
	}
	return &Workflow{
		chatModel:    config.ChatModel,
		callback:     config.Callback,
		systemPrompt: config.SystemPrompt,
	}, nil
}

// Summarize 将旧摘要和旧轮次压缩为新的摘要。
func (w *Workflow) Summarize(ctx context.Context, previousSummary string, turns []session.Turn) (string, error) {
	var content strings.Builder
	content.WriteString("已有摘要：\n")
	if strings.TrimSpace(previousSummary) == "" {
		content.WriteString("无\n")
	} else {
		content.WriteString(previousSummary)
		content.WriteString("\n")
	}
	content.WriteString("\n待压缩对话：\n")
	for i, turn := range turns {
		fmt.Fprintf(&content, "第%d轮\n用户：%s\n助手：%s\n", i+1, turn.User, turn.Assistant)
	}

	msg, err := w.chatModel.Generate(ctx, []*schema.Message{
		schema.SystemMessage(w.systemPrompt),
		schema.UserMessage(content.String()),
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(msg.Content), nil
}
