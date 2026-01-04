package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/lambdasawa/oob-probe-mcp-server/internal/oob"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	oob.SendDesktopNotification("ping")

	mgr := oob.NewListenerManager()
	defer mgr.CloseAll()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		mgr.CloseAll()
	}()

	if err := oob.NewMCPServer(mgr).Run(ctx, &mcp.StdioTransport{}); err != nil {
		panic(err)
	}
}
