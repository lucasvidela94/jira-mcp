package main

import (
	"errors"
	"testing"
)

type fakeServer struct {
	serveStdioCalled bool
	serveSSECalled   bool
	serveSSEAddr     string
	returnErr        error
}

func (f *fakeServer) ServeStdio() error {
	f.serveStdioCalled = true
	return f.returnErr
}

func (f *fakeServer) ServeSSE(addr string) error {
	f.serveSSECalled = true
	f.serveSSEAddr = addr
	return f.returnErr
}

func TestDispatchTransport_DefaultsToStdio(t *testing.T) {
	server := &fakeServer{}
	if err := dispatchTransport(server, "stdio", "8080"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !server.serveStdioCalled {
		t.Error("expected ServeStdio to be called")
	}
	if server.serveSSECalled {
		t.Error("expected ServeSSE not to be called")
	}
}

func TestDispatchTransport_HTTP(t *testing.T) {
	server := &fakeServer{}
	if err := dispatchTransport(server, "http", "9090"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if server.serveStdioCalled {
		t.Error("expected ServeStdio not to be called")
	}
	if !server.serveSSECalled {
		t.Fatal("expected ServeSSE to be called")
	}
	if server.serveSSEAddr != ":9090" {
		t.Errorf("expected address :9090, got %s", server.serveSSEAddr)
	}
}

func TestDispatchTransport_InvalidTransport(t *testing.T) {
	server := &fakeServer{}
	err := dispatchTransport(server, "ws", "8080")
	if err == nil {
		t.Fatal("expected error")
	}
	if server.serveStdioCalled || server.serveSSECalled {
		t.Error("expected no server method to be called")
	}
}

func TestDispatchTransport_PropagatesError(t *testing.T) {
	want := errors.New("boom")
	server := &fakeServer{returnErr: want}

	err := dispatchTransport(server, "stdio", "8080")
	if !errors.Is(err, want) {
		t.Errorf("expected error %v, got %v", want, err)
	}
}
