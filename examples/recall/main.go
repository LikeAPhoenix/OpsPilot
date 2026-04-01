package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"OpsPilot/internal/bootstrap"

	"go.uber.org/zap"
)

// main 演示如何直接调用检索器召回知识库内容。
func main() {
	ctx := context.Background()

	app, err := bootstrap.New(ctx, "")
	if err != nil {
		log.Fatalf("bootstrap app: %v", err)
	}
	defer app.Close()
	logger := app.Logger.Named("examples.recall")

	query := "服务下线是什么原因"
	if len(os.Args) > 1 && strings.TrimSpace(os.Args[1]) != "" {
		query = strings.Join(os.Args[1:], " ")
	}

	docs, err := app.Retriever.Retrieve(ctx, query)
	if err != nil {
		logger.Fatal("retrieve failed", zap.String("query", query), zap.Error(err))
	}

	fmt.Printf("Q: %s\n", query)
	for i, doc := range docs {
		fmt.Printf("A%d: %s\n", i+1, doc.Content)
	}
	fmt.Printf("Done %d\n", len(docs))
}
