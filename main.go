package main

import (
	"fmt"
	"log"
	"os"

	"github.com/lucasvidela/jira-mcp/internal/config"
	"github.com/lucasvidela/jira-mcp/internal/jira"
	"github.com/lucasvidela/jira-mcp/internal/mcp"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %s\n", err)
		os.Exit(1)
	}

	client := jira.New(cfg, nil)
	server := mcp.NewServer(client)

	if err := server.ServeStdio(); err != nil {
		log.Fatalf("server error: %s", err)
	}
}
