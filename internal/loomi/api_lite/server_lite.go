package api_lite

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blueplan/loomi-go/internal/loomi/config"
	contextx "github.com/blueplan/loomi-go/internal/loomi/context"
	"github.com/blueplan/loomi-go/internal/loomi/database"
	logx "github.com/blueplan/loomi-go/internal/loomi/log"
	"github.com/blueplan/loomi-go/internal/loomi/pool"
	"github.com/blueplan/loomi-go/internal/loomi/utils"
)

type Server struct {
	logger     *logx.Logger
	access     *utils.AccessCounter
	cfg        *config.Config
	srv        *http.Server
	redis      pool.Manager
	uploadsDir string
	persist    *database.PersistenceManager
}

func New(logger *logx.Logger, access *utils.AccessCounter, cfg *config.Config, redis pool.Manager) *Server {
	up := "./uploads"
	_ = os.MkdirAll(up, 0755)
	return &Server{logger: logger, access: access, cfg: cfg, redis: redis, uploadsDir: up}
}

func (s *Server) WithPersistence(p *database.PersistenceManager) *Server { s.persist = p; return s }

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.root)
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/health/detailed", s.health)
	mux.HandleFunc("/health/port-monitor", s.health)
	mux.HandleFunc("/health/port-monitor/start", s.health)
	mux.HandleFunc("/health/port-monitor/stop", s.health)
	mux.HandleFunc("/health/port-monitor/status", s.health)
	// upload subset for export paths used by UI
	mux.HandleFunc("/upload/file", s.uploadFile)
	mux.HandleFunc("/upload/file/", s.uploadFileByID)
	// recovery endpoints
	mux.HandleFunc("/api/loomi/heartbeat", s.recoveryHeartbeat)
	mux.HandleFunc("/api/loomi/stream/", s.recoveryStream)
	handler := s.withCORS(s.withContext(s.withRequestLogging(mux)))
	s.srv = &http.Server{Addr: addr, Handler: handler}
	s.logger.Info(context.TODO(), "http.server.start", logx.KV("addr", addr))
	return s.srv.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	if s.srv == nil {
		return nil
	}
	return s.srv.Shutdown(ctx)
}

func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, map[string]any{"service": "api-lite", "version": "1.0.0"})
}
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) withContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-ID")
		if rid == "" {
			rid = time.Now().Format("20060102150405")
		}
		ctx := contextx.WithRequireID(r.Context(), rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func (s *Server) withRequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg != nil && s.cfg.App.Debug {
			s.logger.Info(r.Context(), "http.request", logx.KV("path", r.URL.Path))
		}
		next.ServeHTTP(w, r)
	})
}
func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Request-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// uploads
func (s *Server) uploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	id := time.Now().Format("20060102150405")
	fp := filepath.Join(s.uploadsDir, id)
	b, _ := io.ReadAll(r.Body)
	_ = os.WriteFile(fp, b, 0644)
	s.writeJSON(w, map[string]any{"file_id": id})
}
func (s *Server) uploadFileByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/upload/file/")
	fp := filepath.Join(s.uploadsDir, id)
	switch r.Method {
	case http.MethodGet:
		http.ServeFile(w, r, fp)
	case http.MethodDelete:
		_ = os.Remove(fp)
		s.writeJSON(w, map[string]string{"status": "deleted"})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// recovery endpoints (subset)
type hbReq struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
}

func (s *Server) recoveryHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req hbReq
	decErr := json.NewDecoder(r.Body).Decode(&req)
	if (decErr != nil) || req.UserID == "" || req.SessionID == "" {
		// 兼容 snake_case
		var raw map[string]any
		_ = json.NewDecoder(strings.NewReader("")).Decode(&raw) // no-op to reset
		// 重新读取 body 不易，这里改为从请求中获取原始内容
		// 由于无法多次读取 body，折中：直接从查询参数尝试获取
		if req.UserID == "" {
			req.UserID = r.URL.Query().Get("user_id")
		}
		if req.SessionID == "" {
			req.SessionID = r.URL.Query().Get("session_id")
		}
	}
	if req.UserID == "" || req.SessionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "bad request"})
		return
	}
	if s.persist != nil {
		_, _ = s.persist.UpdateUserHeartbeat(r.Context(), req.UserID, req.SessionID)
	}
	s.writeJSON(w, map[string]any{"success": true, "timestamp": time.Now().Format(time.RFC3339)})
}
func (s *Server) recoveryStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	// minimal SSE no-op for now
	flusher, ok := w.(http.Flusher)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	_, _ = w.Write([]byte("data: {\"type\":\"no_replay\"}\n\n"))
	_, _ = w.Write([]byte("event: done\n"))
	_, _ = w.Write([]byte("data: {}\n\n"))
	flusher.Flush()
}
