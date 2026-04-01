package session

import (
	"context"
	"errors"
	"time"

	"github.com/cloudwego/eino/schema"
)

var (
	ErrSessionNotFound  = errors.New("会话不存在")
	ErrSessionForbidden = errors.New("无权访问该会话")
	ErrSessionConflict  = errors.New("会话处理中，请稍后重试")
)

// Turn 表示一轮完整问答。
type Turn struct {
	User      string `json:"user"`
	Assistant string `json:"assistant"`
}

// Snapshot 表示当前会话在存储中的短期记忆状态。
type Snapshot struct {
	SessionID string
	UserID    string
	Summary   string
	Turns     []Turn
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UnlockFunc 表示分布式锁的释放函数。
type UnlockFunc func(ctx context.Context) error

// Repository 定义会话短期记忆的最小读写能力。
type Repository interface {
	Create(ctx context.Context, sessionID, userID string) error
	Get(ctx context.Context, sessionID string) (*Snapshot, error)
	AppendTurn(ctx context.Context, sessionID string, turn Turn) (*Snapshot, error)
	ReplaceSummaryAndTrim(ctx context.Context, sessionID, summary string, keepLast int) error
	AcquireLock(ctx context.Context, sessionID string) (UnlockFunc, error)
}

// ToMessages 将摘要和最近轮次转换为聊天工作流可消费的历史消息。
func ToMessages(snapshot *Snapshot) []*schema.Message {
	if snapshot == nil {
		return nil
	}

	var history []*schema.Message
	if snapshot.Summary != "" {
		history = append(history, schema.SystemMessage("以下是当前会话的历史摘要：\n"+snapshot.Summary))
	}
	for _, turn := range snapshot.Turns {
		if turn.User != "" {
			history = append(history, schema.UserMessage(turn.User))
		}
		if turn.Assistant != "" {
			history = append(history, schema.AssistantMessage(turn.Assistant, nil))
		}
	}
	return history
}
