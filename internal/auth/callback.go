package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// CallbackServer listens on 127.0.0.1:0 for the OAuth redirect callback.
type CallbackServer struct {
	server       *http.Server
	port         int
	code         chan string
	err          chan error
	ctx          context.Context
	cancel       context.CancelFunc
	BrowserOpen  func(url string) error // injectable for testing
}

// NewCallbackServer creates a new callback server that hasn't started yet.
func NewCallbackServer() *CallbackServer {
	return &CallbackServer{
		code:        make(chan string, 1),
		err:         make(chan error, 1),
		BrowserOpen: openBrowser,
	}
}

// Start begins listening on 127.0.0.1:0 with the given timeout.
// The timeout limits how long the server will wait for a callback.
func (s *CallbackServer) Start(timeout time.Duration) {
	s.ctx, s.cancel = context.WithTimeout(context.Background(), timeout)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		s.port = 0
		s.server = &http.Server{}
		return
	}
	s.port = listener.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", s.handleCallback)

	s.server = &http.Server{
		Handler: mux,
	}

	go func() {
		_ = s.server.Serve(listener)
	}()
}

// Stop gracefully shuts down the callback server.
func (s *CallbackServer) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.server.Shutdown(ctx)
	}
}

// Port returns the port the server is listening on (0 if not started).
func (s *CallbackServer) Port() int {
	return s.port
}

// WaitForCode blocks until the callback is received, an error occurs, or the timeout expires.
func (s *CallbackServer) WaitForCode() (string, error) {
	if s.ctx == nil {
		return "", fmt.Errorf("server not started")
	}

	select {
	case <-s.ctx.Done():
		return "", fmt.Errorf("callback timeout: no authorization received within the time limit")
	case err := <-s.err:
		return "", err
	case code := <-s.code:
		if code == "" {
			return "", fmt.Errorf("callback received empty authorization code")
		}
		return code, nil
	}
}

func (s *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	errParam := r.URL.Query().Get("error")
	if errParam != "" {
		desc := r.URL.Query().Get("error_description")
		if desc == "" {
			desc = "authorization denied"
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Authorization failed: %s", desc)

		select {
		case s.err <- fmt.Errorf("oauth error: %s — %s", errParam, desc):
		default:
		}
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Missing authorization code")

		select {
		case s.err <- fmt.Errorf("callback received no authorization code"):
		default:
		}
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "<html><body><h1>Authorization successful!</h1><p>You can close this window.</p></body></html>")

	select {
	case s.code <- code:
	default:
	}
}

// OpenBrowser attempts to open the given URL in the default browser.
// Uses the injectable BrowserOpen field; defaults to platform-specific opener.
func (s *CallbackServer) OpenBrowser(url string) error {
	if s.BrowserOpen == nil {
		return fmt.Errorf("browser open not configured")
	}
	return s.BrowserOpen(url)
}
