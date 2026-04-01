package snowflake

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

const (
	nodeBits     = 10
	sequenceBits = 12

	maxNodeID   = -1 ^ (-1 << nodeBits)
	maxSequence = -1 ^ (-1 << sequenceBits)

	nodeShift      = sequenceBits
	timestampShift = sequenceBits + nodeBits
)

var epoch = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

// Generator 使用 Snowflake 规则生成单机内有序的唯一 ID。
type Generator struct {
	mu            sync.Mutex
	nodeID        int64
	lastTimestamp int64
	sequence      int64
}

// NewGenerator 创建指定节点的 Snowflake 生成器。
func NewGenerator(nodeID int64) (*Generator, error) {
	if nodeID < 0 || nodeID > maxNodeID {
		return nil, fmt.Errorf("invalid snowflake node id %d", nodeID)
	}
	return &Generator{nodeID: nodeID}, nil
}

// NextInt64 返回下一个递增 ID。
func (g *Generator) NextInt64() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := currentMillis()
	if now < g.lastTimestamp {
		now = g.lastTimestamp
	}

	if now == g.lastTimestamp {
		g.sequence = (g.sequence + 1) & maxSequence
		if g.sequence == 0 {
			now = waitNextMillis(g.lastTimestamp)
		}
	} else {
		g.sequence = 0
	}

	g.lastTimestamp = now
	return ((now - epoch) << timestampShift) | (g.nodeID << nodeShift) | g.sequence
}

// NextString 返回十进制字符串 ID，便于直接作为 JSON 和路由参数使用。
func (g *Generator) NextString() string {
	return strconv.FormatInt(g.NextInt64(), 10)
}

func currentMillis() int64 {
	return time.Now().UnixMilli()
}

func waitNextMillis(last int64) int64 {
	now := currentMillis()
	for now <= last {
		time.Sleep(time.Millisecond)
		now = currentMillis()
	}
	return now
}
