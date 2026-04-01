package chat

import (
	"context"
	"errors"
	"strings"

	"OpsPilot/internal/domain/session"

	"github.com/cloudwego/eino/schema"
)

var (
	ErrUnauthorized = errors.New("未认证用户")
	ErrForbidden    = session.ErrSessionForbidden
	ErrConflict     = session.ErrSessionConflict
	ErrNotFound     = session.ErrSessionNotFound
)

// Agent 定义聊天工作流暴露的同步和流式能力。
type Agent interface {
	Chat(ctx context.Context, query string, history []*schema.Message) (string, error)
	Stream(ctx context.Context, query string, history []*schema.Message, send func(string) error) (string, error)
}

// StreamSender 抽象流式输出端，HTTP SSE 或其他传输层都可以复用。
type StreamSender interface {
	Send(eventType, data string) error
}

// Summarizer 定义会话摘要压缩能力。
type Summarizer interface {
	Summarize(ctx context.Context, previousSummary string, turns []session.Turn) (string, error)
}

// IDGenerator 定义会话 ID 生成能力。
type IDGenerator interface {
	NextString() string
}

// Config 定义短期记忆的窗口策略。
type Config struct {
	RetainTurns           int
	CompactThresholdTurns int
}

// Service 负责拼接会话上下文，并维护问答消息窗口。
type Service struct {
	sessions           session.Repository
	agent              Agent
	summarizer         Summarizer
	sessionIDGenerator IDGenerator
	retainTurns        int
	compactThreshold   int
}

// Result 表示一次对话的最终回答。
type Result struct {
	Answer    string
	SessionID string
}

// NewService 创建聊天应用服务。
func NewService(sessions session.Repository, agent Agent, summarizer Summarizer, sessionIDGenerator IDGenerator, config Config) *Service {
	if config.RetainTurns <= 0 {
		config.RetainTurns = 6
	}
	if config.CompactThresholdTurns <= 0 {
		config.CompactThresholdTurns = 9
	}
	return &Service{
		sessions:           sessions,
		agent:              agent,
		summarizer:         summarizer,
		sessionIDGenerator: sessionIDGenerator,
		retainTurns:        config.RetainTurns,
		compactThreshold:   config.CompactThresholdTurns,
	}
}

// Chat 使用历史消息构造上下文，并在成功后写回本轮问答。
func (s *Service) Chat(ctx context.Context, userID, id, msg string) (*Result, error) {
	sessionID, snapshot, unlock, err := s.prepareSession(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	defer unlock()

	answer, err := s.agent.Chat(ctx, msg, session.ToMessages(snapshot))
	if err != nil {
		return nil, err
	}

	if err := s.appendAndCompact(ctx, sessionID, msg, answer); err != nil {
		return nil, err
	}
	return &Result{
		Answer:    answer,
		SessionID: sessionID,
	}, nil
}

// Stream 以增量输出方式返回回答，并在结束后补写完整问答到会话存储。
func (s *Service) Stream(ctx context.Context, userID, id, msg string, sender StreamSender) error {
	sessionID, snapshot, unlock, err := s.prepareSession(ctx, userID, id)
	if err != nil {
		return writeStreamError(sender, err)
	}
	defer unlock()

	if err := sender.Send("session", sessionID); err != nil {
		return err
	}

	answer, err := s.agent.Stream(ctx, msg, session.ToMessages(snapshot), func(chunk string) error {
		return sender.Send("message", chunk)
	})
	if err != nil {
		return writeStreamError(sender, err)
	}

	if err := s.appendAndCompact(ctx, sessionID, msg, answer); err != nil {
		return writeStreamError(sender, err)
	}

	// done 事件是前端关闭流式消费的显式信号。
	if err := sender.Send("done", "[DONE]"); err != nil {
		return err
	}
	return nil
}

// writeStreamError 在流式调用失败时尽量将错误回传给客户端，再返回原始错误。
func writeStreamError(sender StreamSender, err error) error {
	if sender != nil {
		_ = sender.Send("message", "处理请求失败: "+err.Error())
		_ = sender.Send("done", "[DONE]")
	}
	return err
}

func (s *Service) prepareSession(ctx context.Context, userID, id string) (string, *session.Snapshot, func(), error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", nil, nil, ErrUnauthorized
	}

	sessionID := strings.TrimSpace(id)
	if sessionID == "" {
		sessionID = s.sessionIDGenerator.NextString()
	}

	unlockFunc, err := s.sessions.AcquireLock(ctx, sessionID)
	if err != nil {
		return "", nil, nil, err
	}

	release := func() {
		_ = unlockFunc(context.Background())
	}

	if strings.TrimSpace(id) == "" {
		if err := s.sessions.Create(ctx, sessionID, userID); err != nil {
			release()
			return "", nil, nil, err
		}
	}

	snapshot, err := s.sessions.Get(ctx, sessionID)
	if err != nil {
		release()
		return "", nil, nil, err
	}
	if snapshot.UserID != userID {
		release()
		return "", nil, nil, ErrForbidden
	}

	return sessionID, snapshot, release, nil
}

func (s *Service) appendAndCompact(ctx context.Context, sessionID string, userMessage, answer string) error {
	updated, err := s.sessions.AppendTurn(ctx, sessionID, session.Turn{
		User:      userMessage,
		Assistant: answer,
	})
	if err != nil {
		return err
	}

	if len(updated.Turns) <= s.compactThreshold {
		return nil
	}

	trimCount := len(updated.Turns) - s.retainTurns
	if trimCount <= 0 {
		return nil
	}

	newSummary, err := s.summarizer.Summarize(ctx, updated.Summary, updated.Turns[:trimCount])
	if err != nil {
		return err
	}

	return s.sessions.ReplaceSummaryAndTrim(ctx, sessionID, newSummary, s.retainTurns)
}
