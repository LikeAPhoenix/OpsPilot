package sse

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Client 封装 SSE 输出能力，负责统一事件格式和并发写保护。
type Client struct {
	id      string
	writer  http.ResponseWriter
	flusher http.Flusher
	mu      sync.Mutex
}

// NewClient 根据 HTTP ResponseWriter 创建 SSE 客户端，并发送连接建立事件。
func NewClient(w http.ResponseWriter, clientID string) (*Client, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming is not supported by this server")
	}

	headers := w.Header()
	headers.Set("Content-Type", "text/event-stream")
	headers.Set("Cache-Control", "no-cache")
	headers.Set("Connection", "keep-alive")
	headers.Set("Access-Control-Allow-Origin", "*")
	// 关闭 Nginx 代理缓冲，确保分片内容可以及时刷新到客户端。
	headers.Set("X-Accel-Buffering", "no")

	client := &Client{
		id:      clientID,
		writer:  w,
		flusher: flusher,
	}

	if err := client.Send("connected", fmt.Sprintf(`{"status":"connected","client_id":"%s"}`, client.id)); err != nil {
		return nil, err
	}

	return client, nil
}

// Send 发送一个 SSE 事件，并按协议拆分多行数据。
func (c *Client) Send(eventType, data string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, err := fmt.Fprintf(c.writer, "id: %d\nevent: %s\n", time.Now().UnixNano(), eventType); err != nil {
		return err
	}

	// SSE 要求每一行数据都以独立的 data: 前缀输出，因此先统一换行格式。
	normalized := strings.ReplaceAll(data, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	for _, line := range strings.Split(normalized, "\n") {
		if _, err := fmt.Fprintf(c.writer, "data: %s\n", line); err != nil {
			return err
		}
	}

	if _, err := io.WriteString(c.writer, "\n"); err != nil {
		return err
	}

	c.flusher.Flush()
	return nil
}
