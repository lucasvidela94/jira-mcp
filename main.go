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

type runner interface {
	ServeStdio() error
	ServeSSE(addr string) error
}

func main() {
	transport := flag.String("transport", "stdio", "transport type: stdio or http")
	port := flag.String("port", "8080", "HTTP listen port (used with --transport http)")
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

	if err := dispatchTransport(server, *transport, *port); err != nil {
		log.Fatalf("server error: %s", err)
	}
}

func dispatchTransport(server runner, transport, port string) error {
	switch transport {
	case "http":
		addr := fmt.Sprintf(":%s", port)
		return server.ServeSSE(addr)
	case "stdio":
		return server.ServeStdio()
	default:
		return fmt.Errorf("invalid transport: %s (must be stdio or http)", transport)
	}
}
