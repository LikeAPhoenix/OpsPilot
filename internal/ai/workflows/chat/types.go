package chat

import "github.com/cloudwego/eino/schema"

// Input 表示聊天工作流的输入内容。
type Input struct {
	ID      string
	Query   string
	History []*schema.Message
}
