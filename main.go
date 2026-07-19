package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"github.com/lucasvidela94/jira-mcp/internal/config"
	"github.com/lucasvidela94/jira-mcp/internal/jira"
	"github.com/lucasvidela94/jira-mcp/internal/mcp"
	"github.com/lucasvidela94/jira-mcp/internal/update"
)

// version is set by goreleaser at build time.
var version = "dev"

// resolveVersion returns the embedded version, or falls back to Go module build info.
func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	return version
}

type runner interface {
	ServeStdio() error
	ServeSSE(addr string) error
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "update" {
		if err := runUpdate(); err != nil {
			log.Fatalf("update failed: %s", err)
		}
		return
	}

	transport := flag.String("transport", "stdio", "transport type: stdio or http")
	port := flag.String("port", "8080", "HTTP listen port (used with --transport http)")
	versionFlag := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Println(resolveVersion())
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

func runUpdate() error {
	u := update.NewSelfUpdater()
	result, err := u.Run(context.Background(), resolveVersion())
	if err != nil {
		return err
	}
	fmt.Println(result.Message)
	if result.Updated {
		fmt.Println("Restart your MCP client to use the new version.")
	}
	return nil
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
