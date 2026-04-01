package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	authapp "OpsPilot/internal/application/auth"
	"OpsPilot/internal/bootstrap"

	"go.uber.org/zap"
)

// main 演示同一会话下连续两轮聊天调用。
func main() {
	ctx := context.Background()

	app, err := bootstrap.New(ctx, "")
	if err != nil {
		log.Fatalf("bootstrap app: %v", err)
	}
	defer app.Close()
	logger := app.Logger.Named("examples.chat")

	user, err := app.AuthService.Register(ctx, "example-user", "example-password")
	if err != nil && !errors.Is(err, authapp.ErrUsernameTaken) {
		logger.Fatal("register example user failed", zap.Error(err))
	}
	userID := ""
	if user != nil {
		userID = user.ID
	}
	if user == nil {
		loginResult, loginErr := app.AuthService.Login(ctx, "example-user", "example-password")
		if loginErr != nil {
			logger.Fatal("login example user failed", zap.Error(loginErr))
		}
		userID = loginResult.UserID
	}

	sessionID := ""
	questions := []string{
		"你好",
		"现在是几点",
	}

	// 复用同一个 sessionID，验证会话历史会被自动带入后续轮次。
	for i, question := range questions {
		result, err := app.ChatService.Chat(ctx, userID, sessionID, question)
		if err != nil {
			logger.Fatal("chat round failed", zap.Int("round", i+1), zap.Error(err))
		}
		sessionID = result.SessionID
		fmt.Printf("Q%d: %s\n", i+1, question)
		fmt.Printf("A%d: %s\n", i+1, result.Answer)
		if i != len(questions)-1 {
			fmt.Println("-----")
		}
	}
}
