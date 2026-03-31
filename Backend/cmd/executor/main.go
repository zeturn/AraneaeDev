package main

import (
	"context"
	"os/signal"
	"syscall"

	"araneae-go/internal/common"
	"araneae-go/internal/executor"
)

func main() {
	cfg := common.LoadExecutorConfig()
	app, err := executor.NewApp(cfg)
	if err != nil {
		panic(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := app.Run(ctx); err != nil {
		panic(err)
	}
}
