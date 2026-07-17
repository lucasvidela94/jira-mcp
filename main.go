package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/lucasvidela94/jira-mcp/internal/config"
	"github.com/lucasvidela94/jira-mcp/internal/jira"
	"github.com/lucasvidela94/jira-mcp/internal/mcp"
)

// version is set by goreleaser at build time.
var version = "dev"

func main() {
	versionFlag := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}

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
