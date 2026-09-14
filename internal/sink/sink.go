package sink

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
)

type Event struct {
	RunID      string `json:"run_id"`
	Path       string `json:"path"`
	Bytes      int    `json:"bytes"`
	PayloadSHA string `json:"payload_sha256"`
}

type Server struct {
	server        *http.Server
	ln            net.Listener
	advertiseHost string
	mu            sync.RWMutex
	events        []Event
	allowedRuns   map[string]struct{}
}

func Start() (*Server, error) {
	return StartOn("127.0.0.1:0")
}

// StartOn starts the sink on an explicit listener address. The default binds
// loopback; an outer VM runner can pass its private bridge address explicitly.
func StartOn(address string) (*Server, error) {
	if err := validateBindAddress(address); err != nil {
		return nil, err
	}
	advertiseHost := "127.0.0.1"
	if host, _, splitErr := net.SplitHostPort(address); splitErr == nil && host != "" && host != "0.0.0.0" && host != "::" {
		advertiseHost = host
	}
	s := &Server{advertiseHost: advertiseHost, allowedRuns: make(map[string]struct{})}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/events", s.handleEvent)
	mux.HandleFunc("/git-push", s.handleEvent)
	mux.HandleFunc("/redirect", s.handleRedirect)
	mux.HandleFunc("/response", s.handleResponse)
	mux.HandleFunc("/runs/", s.handleRun)
	// Bind on the outer VM interface so a Docker bridge container can reach it.
	// The runner only advertises this address inside the isolated VM.
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	s.ln = ln
	s.server = &http.Server{Handler: mux}
	go s.server.Serve(ln)
	return s, nil
}

func (s *Server) URL() string {
	_, port, _ := net.SplitHostPort(s.ln.Addr().String())
	return "http://" + net.JoinHostPort(s.advertiseHost, port)
}

func (s *Server) ContainerURL() string {
	_, port, _ := net.SplitHostPort(s.ln.Addr().String())
	return "http://" + net.JoinHostPort(s.advertiseHost, port)
}

func validateBindAddress(address string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("sink bind address must be host:port: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("sink bind host must be a literal loopback or private IP")
	}
	if ip.IsUnspecified() {
		return fmt.Errorf("sink must not bind an unspecified or public interface")
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || privateIPv4(ip) || privateIPv6(ip) {
		return nil
	}
	return fmt.Errorf("sink bind host %s is not loopback or private", host)
}

func privateIPv4(ip net.IP) bool {
	ip = ip.To4()
	if ip == nil {
		return false
	}
	return ip[0] == 10 || (ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31) || (ip[0] == 192 && ip[1] == 168)
}

func privateIPv6(ip net.IP) bool {
	ip = ip.To16()
	return ip != nil && (ip[0]&0xfe) == 0xfc
}

func (s *Server) Close() error { return s.server.Close() }

func (s *Server) Events(runID string) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Event, 0)
	for _, event := range s.events {
		if event.RunID == runID {
			result = append(result, event)
		}
	}
	return result
}

// RegisterRun binds accepted sink events to a run created by the coordinator.
// Attackers may know their own run ID, but cannot use the sink as a cross-task
// evidence channel by guessing another valid-looking ID.
func (s *Server) RegisterRun(runID string) error {
	if !validRunID(runID) {
		return fmt.Errorf("invalid run id")
	}
	s.mu.Lock()
	s.allowedRuns[runID] = struct{}{}
	s.mu.Unlock()
	return nil
}

func (s *Server) runAllowed(runID string) bool {
	s.mu.RLock()
	_, ok := s.allowedRuns[runID]
	s.mu.RUnlock()
	return ok
}

func (s *Server) handleEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var data []byte
	if r.Method == http.MethodPost {
		var err error
		data, err = ioutil.ReadAll(http.MaxBytesReader(w, r.Body, 1024*1024))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	runID := r.Header.Get("X-Holdout-Run-ID")
	if runID == "" {
		runID = r.URL.Query().Get("run_id")
	}
	if !validRunID(runID) || !s.runAllowed(runID) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	digest := sha256.Sum256(data)
	s.mu.Lock()
	s.events = append(s.events, Event{RunID: runID, Path: r.URL.Path, Bytes: len(data), PayloadSHA: hex.EncodeToString(digest[:])})
	s.mu.Unlock()
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleRedirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	runID := r.URL.Query().Get("run_id")
	if !validRunID(runID) || !s.runAllowed(runID) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Location", "/events?run_id="+runID)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (s *Server) handleResponse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !validRunID(r.URL.Query().Get("run_id")) || !s.runAllowed(r.URL.Query().Get("run_id")) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte("holdout-synthetic-response"))
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	runID := strings.TrimPrefix(r.URL.Path, "/runs/")
	if !validRunID(runID) || !s.runAllowed(runID) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	events := s.Events(runID)
	if events == nil {
		events = []Event{}
	}
	if err := json.NewEncoder(w).Encode(events); err != nil {
		return
	}
}

func (e Event) String() string { return fmt.Sprintf("%s:%d", e.Path, e.Bytes) }

func validRunID(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.' || char == ':' {
			continue
		}
		return false
	}
	return true
}
