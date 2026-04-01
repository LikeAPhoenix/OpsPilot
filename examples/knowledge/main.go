package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"OpsPilot/internal/bootstrap"

	"go.uber.org/zap"
)

// main 演示对单个文件或整个目录执行知识库索引。
func main() {
	ctx := context.Background()

	app, err := bootstrap.New(ctx, "")
	if err != nil {
		log.Fatalf("bootstrap app: %v", err)
	}
	defer app.Close()
	logger := app.Logger.Named("examples.knowledge")

	targetPath := "./docs/knowledge"
	if len(os.Args) > 1 && strings.TrimSpace(os.Args[1]) != "" {
		targetPath = os.Args[1]
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		logger.Fatal("stat target path failed", zap.String("path", targetPath), zap.Error(err))
	}

	if !info.IsDir() {
		indexFile(ctx, app, targetPath)
		return
	}

	err = filepath.WalkDir(targetPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".md") {
			fmt.Printf("[skip] not a markdown file: %s\n", path)
			return nil
		}
		indexFile(ctx, app, path)
		return nil
	})
	if err != nil {
		logger.Fatal("walk target path failed", zap.String("path", targetPath), zap.Error(err))
	}
}

// indexFile 对单个 Markdown 文件执行知识索引。
func indexFile(ctx context.Context, app *bootstrap.App, path string) {
	logger := app.Logger.Named("examples.knowledge")
	if !strings.EqualFold(filepath.Ext(path), ".md") {
		logger.Fatal("only markdown files are supported", zap.String("path", path))
	}

	fmt.Printf("[start] indexing file: %s\n", path)
	if err := app.KnowledgeService.IndexPath(ctx, path); err != nil {
		logger.Fatal("index file failed", zap.String("path", path), zap.Error(err))
	}
	fmt.Printf("[done] indexing file: %s\n", path)
}
