package main

import (
	"context"
	"log"

	"OpsPilot/internal/bootstrap"

	"go.uber.org/zap"
)

// main 启动 HTTP 服务，并输出最终监听地址。
func main() {
	app, err := bootstrap.New(context.Background(), "")
	if err != nil {
		log.Fatalf("bootstrap app: %v", err)
	}
	defer app.Close()

	app.Logger.Info("starting server", zap.String("address", app.Config.Server.Address))
	if err := app.Engine.Run(app.Config.Server.Address); err != nil {
		app.Logger.Fatal("start server", zap.Error(err))
	}
}
