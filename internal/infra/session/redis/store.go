package redis

import (
	"OpsPilot/internal/domain/session"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	metaKeyPrefix = "opspilot:session:meta:"  // 存会话元数据
	turnKeyPrefix = "opspilot:session:turns:"  // 存每一轮问答明细
	lockKeyPrefix = "opspilot:session:lock:"
	lockTokenTTL  = 30 * time.Second
)

var unlockScript = goredis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
  return redis.call("del", KEYS[1])
end
return 0
`)

// Store 使用 Redis 保存会话短期记忆和会话锁。
type Store struct {
	client *goredis.Client
	ttl    time.Duration
}

type sessionMeta struct {
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	Summary   string    `json:"summary"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewStore 创建 Redis 会话仓储。
func NewStore(client *goredis.Client, ttl time.Duration) *Store {
	return &Store{
		client: client,
		ttl:    ttl,
	}
}

// Create 创建一个新的会话元数据。
func (s *Store) Create(ctx context.Context, sessionID, userID string) error {
	now := time.Now()
	meta := sessionMeta{
		SessionID: sessionID,
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	ok, err := s.client.SetNX(ctx, metaKey(sessionID), data, s.ttl).Result()
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	return s.client.Expire(ctx, turnKey(sessionID), s.ttl).Err()
}

// Get 返回指定会话的完整快照。
func (s *Store) Get(ctx context.Context, sessionID string) (*session.Snapshot, error) {
	meta, err := s.getMeta(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	encodedTurns, err := s.client.LRange(ctx, turnKey(sessionID), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	turns := make([]session.Turn, 0, len(encodedTurns))
	for _, encoded := range encodedTurns {
		var turn session.Turn
		if err := json.Unmarshal([]byte(encoded), &turn); err != nil {
			return nil, err
		}
		turns = append(turns, turn)
	}

	if err := s.refreshTTL(ctx, sessionID); err != nil {
		return nil, err
	}

	return &session.Snapshot{
		SessionID: meta.SessionID,
		UserID:    meta.UserID,
		Summary:   meta.Summary,
		Turns:     turns,
		CreatedAt: meta.CreatedAt,
		UpdatedAt: meta.UpdatedAt,
	}, nil
}

// AppendTurn 追加一轮完整问答，并返回最新快照。
func (s *Store) AppendTurn(ctx context.Context, sessionID string, turn session.Turn) (*session.Snapshot, error) {
	meta, err := s.getMeta(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	encodedTurn, err := json.Marshal(turn)
	if err != nil {
		return nil, err
	}

	meta.UpdatedAt = time.Now()
	encodedMeta, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}

	pipe := s.client.TxPipeline()
	pipe.RPush(ctx, turnKey(sessionID), string(encodedTurn))
	pipe.Set(ctx, metaKey(sessionID), encodedMeta, s.ttl)
	pipe.Expire(ctx, turnKey(sessionID), s.ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}

	return s.Get(ctx, sessionID)
}

// ReplaceSummaryAndTrim 更新摘要并裁剪最近轮次。
func (s *Store) ReplaceSummaryAndTrim(ctx context.Context, sessionID, summary string, keepLast int) error {
	meta, err := s.getMeta(ctx, sessionID)
	if err != nil {
		return err
	}

	meta.Summary = strings.TrimSpace(summary)
	meta.UpdatedAt = time.Now()
	encodedMeta, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	totalTurns, err := s.client.LLen(ctx, turnKey(sessionID)).Result()
	if err != nil {
		return err
	}
	trimStart := totalTurns - int64(keepLast)
	if trimStart < 0 {
		trimStart = 0
	}

	pipe := s.client.TxPipeline()
	pipe.Set(ctx, metaKey(sessionID), encodedMeta, s.ttl)
	pipe.LTrim(ctx, turnKey(sessionID), trimStart, -1)
	pipe.Expire(ctx, turnKey(sessionID), s.ttl)
	_, err = pipe.Exec(ctx)
	return err
}

// AcquireLock 获取指定会话的短期锁。
func (s *Store) AcquireLock(ctx context.Context, sessionID string) (session.UnlockFunc, error) {
	token := fmt.Sprintf("%d", time.Now().UnixNano())
	ok, err := s.client.SetNX(ctx, lockKey(sessionID), token, lockTokenTTL).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, session.ErrSessionConflict
	}

	return func(ctx context.Context) error {
		_, err := unlockScript.Run(ctx, s.client, []string{lockKey(sessionID)}, token).Result()
		return err
	}, nil
}

func (s *Store) getMeta(ctx context.Context, sessionID string) (*sessionMeta, error) {
	data, err := s.client.Get(ctx, metaKey(sessionID)).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, session.ErrSessionNotFound
		}
		return nil, err
	}

	var meta sessionMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (s *Store) refreshTTL(ctx context.Context, sessionID string) error {
	pipe := s.client.TxPipeline()
	pipe.Expire(ctx, metaKey(sessionID), s.ttl)
	pipe.Expire(ctx, turnKey(sessionID), s.ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func metaKey(sessionID string) string {
	return metaKeyPrefix + sessionID
}

func turnKey(sessionID string) string {
	return turnKeyPrefix + sessionID
}

func lockKey(sessionID string) string {
	return lockKeyPrefix + sessionID
}
