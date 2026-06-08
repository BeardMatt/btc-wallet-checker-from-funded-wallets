package coordinator

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"btcfind/cluster"
	"btcfind/funded"
)

type apiError struct {
	msg  string
	code int
}

func (e *apiError) Error() string { return e.msg }

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) mountAPI(mux *http.ServeMux) {
	regLim := newRateLimiter(s.cfg.RateRegister, time.Minute)
	if s.cfg.RateRegister == 0 {
		regLim = newRateLimiter(10, time.Minute)
	}
	cacheLim := newRateLimiter(s.cfg.RateCache, time.Hour)
	if s.cfg.RateCache == 0 {
		cacheLim = newRateLimiter(2, time.Hour)
	}
	apiLim := newRateLimiter(s.cfg.RateAPI, time.Minute)
	if s.cfg.RateAPI == 0 {
		apiLim = newRateLimiter(120, time.Minute)
	}

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	api := http.NewServeMux()
	api.HandleFunc("/register", s.rateLimit(regLim, s.handleRegister))
	api.HandleFunc("/heartbeat", s.rateLimit(apiLim, s.handleHeartbeat))
	api.HandleFunc("/run", s.rateLimit(apiLim, s.handleRun))
	api.HandleFunc("/cache", s.rateLimit(cacheLim, s.handleCache))
	api.HandleFunc("/stats", s.rateLimit(apiLim, s.handleStats))
	api.HandleFunc("/hit", s.rateLimit(apiLim, s.handleHitHTTP))

	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", bearerAuth(s.cfg.AuthToken, api)))
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.cfg.MaxWorkers > 0 {
		s.mu.RLock()
		full := len(s.workers) >= s.cfg.MaxWorkers
		s.mu.RUnlock()
		if full {
			http.Error(w, "cluster full", http.StatusServiceUnavailable)
			return
		}
	}
	var req cluster.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Hostname == "" {
		req.Hostname = clientIP(r)
	}
	if req.Threads < 1 {
		req.Threads = 1
	}
	resp := s.register(req)
	if resp.WorkerID == "" {
		http.Error(w, "cluster full", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req cluster.HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := s.heartbeat(req.WorkerID, req.State); err != nil {
		var ae *apiError
		if errors.As(err, &ae) {
			http.Error(w, ae.msg, ae.code)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.runStatus())
}

func (s *Server) handleCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	etag, err := funded.CacheETag(s.cachePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if inm := r.Header.Get("If-None-Match"); inm == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	f, err := os.Open(s.cachePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("ETag", etag)
	w.Header().Set("X-Cache-Min-Balance", strconv.FormatUint(s.cfg.MinBalance, 10))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	_, _ = io.Copy(w, f)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req cluster.StatsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	s.recordStats(req)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleHitHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req cluster.HitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := s.handleHit(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}