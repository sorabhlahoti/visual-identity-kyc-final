package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"visual-kyc/api/internal/config"
	"visual-kyc/api/internal/domain"
	"visual-kyc/api/internal/metrics"
	"visual-kyc/api/internal/security"
	"visual-kyc/api/internal/services"
)

type Router struct {
	cfg      config.Config
	svc      *services.AsyncService
	limiter  *RateLimiter
	counters *metrics.Counters
}

func NewRouter(cfg config.Config, svc *services.AsyncService, counters *metrics.Counters) http.Handler {
	r := &Router{cfg: cfg, svc: svc, limiter: NewRateLimiter(120, time.Minute), counters: counters}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", r.health)
	mux.HandleFunc("/ready", r.health)
	mux.HandleFunc("/metrics", r.metrics)
	mux.HandleFunc("/auth/token", r.token)
	mux.HandleFunc("/openapi.yaml", r.openapi)
	mux.HandleFunc("/kyc/enroll", r.withPOST(r.auth(r.rateLimit(r.enroll))))
	mux.HandleFunc("/kyc/verify", r.withPOST(r.auth(r.rateLimit(r.verify))))
	mux.HandleFunc("/kyc/status/", r.auth(r.rateLimit(r.status)))
	mux.HandleFunc("/", r.notFound)
	return corsMiddleware(r.cfg, securityHeaders(recoverMiddleware(mux)))
}

func (r *Router) openapi(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	http.ServeFile(w, req, "openapi.yaml")
}

func (r *Router) health(w http.ResponseWriter, req *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": r.cfg.ServiceName, "mode": "async"})
}

func (r *Router) metrics(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = w.Write([]byte(r.counters.Prometheus()))
}

func (r *Router) token(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	token, err := security.SignJWT(r.cfg.JWTSecret, "local-demo-client", 24*time.Hour)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token, "type": "Bearer"})
}

func (r *Router) enroll(w http.ResponseWriter, req *http.Request) {
	r.counters.IncRequests()
	input, err := r.parseMultipart(w, req)
	if err != nil {
		r.counters.IncErrors()
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	resp, err := r.svc.Enroll(req.Context(), input)
	if err != nil {
		r.counters.IncErrors()
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	r.counters.IncAccepted()
	writeJSON(w, http.StatusAccepted, resp)
}

func (r *Router) verify(w http.ResponseWriter, req *http.Request) {
	r.counters.IncRequests()
	input, err := r.parseMultipart(w, req)
	if err != nil {
		r.counters.IncErrors()
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	resp, err := r.svc.Verify(req.Context(), input)
	if err != nil {
		r.counters.IncErrors()
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	r.counters.IncAccepted()
	writeJSON(w, http.StatusAccepted, resp)
}

func (r *Router) status(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	transactionID := strings.TrimPrefix(req.URL.Path, "/kyc/status/")
	if transactionID == "" || transactionID == req.URL.Path {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "transaction_id is required"})
		return
	}
	rec, err := r.svc.Status(transactionID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "transaction not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (r *Router) parseMultipart(w http.ResponseWriter, req *http.Request) (domain.KYCInput, error) {
	// Hard-limit the whole multipart body. Without this, a very large file can
	// cause the handler to spend time/memory before returning an error.
	req.Body = http.MaxBytesReader(w, req.Body, r.cfg.MaxImageBytes+1024*1024)
	if err := req.ParseMultipartForm(r.cfg.MaxImageBytes); err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			return domain.KYCInput{}, errors.New("request body too large; reduce image size or increase MAX_IMAGE_BYTES")
		}
		return domain.KYCInput{}, err
	}
	file, _, err := req.FormFile("image")
	if err != nil {
		return domain.KYCInput{}, errors.New("multipart field 'image' is required")
	}
	defer file.Close()
	imageBytes, err := io.ReadAll(io.LimitReader(file, r.cfg.MaxImageBytes+1))
	if err != nil {
		return domain.KYCInput{}, err
	}
	if int64(len(imageBytes)) > r.cfg.MaxImageBytes {
		return domain.KYCInput{}, errors.New("image too large")
	}
	return domain.KYCInput{ImageBytes: imageBytes, Name: strings.TrimSpace(req.FormValue("name")), DOB: strings.TrimSpace(req.FormValue("dob")), Gender: strings.TrimSpace(req.FormValue("gender")), CallbackURL: strings.TrimSpace(req.FormValue("callback_url"))}, nil
}

func (r *Router) withPOST(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		next(w, req)
	}
}

func (r *Router) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if !r.cfg.AuthRequired {
			next(w, req)
			return
		}
		auth := req.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing Bearer token"})
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if _, err := security.VerifyJWT(r.cfg.JWTSecret, token); err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}
		next(w, req)
	}
}

func (r *Router) rateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ip := req.RemoteAddr
		if forwarded := req.Header.Get("X-Forwarded-For"); forwarded != "" {
			ip = strings.Split(forwarded, ",")[0]
		}
		if !r.limiter.Allow(ip) {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
			return
		}
		next(w, req)
	}
}

func (r *Router) notFound(w http.ResponseWriter, req *http.Request) {
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("PANIC method=%s path=%s remote=%s panic=%v\n%s", r.Method, r.URL.Path, r.RemoteAddr, rec, debug.Stack())
				writeJSON(w, http.StatusInternalServerError, map[string]string{
					"error": "internal server error; check docker compose logs api",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(cfg config.Config, next http.Handler) http.Handler {
	allowedOrigins := parseAllowedOrigins(cfg.CORSAllowedOrigins)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowOrigin := "*"
		if origin != "" {
			if originAllowed(origin, allowedOrigins) {
				allowOrigin = origin
			} else if !originAllowed("*", allowedOrigins) {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "origin is not allowed by CORS"})
				return
			} else {
				allowOrigin = origin
			}
		}
		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, Origin, X-Requested-With")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "600")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func parseAllowedOrigins(raw string) map[string]bool {
	out := map[string]bool{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out[part] = true
		}
	}
	if len(out) == 0 {
		out["*"] = true
	}
	return out
}

func originAllowed(origin string, allowed map[string]bool) bool {
	if allowed["*"] || allowed[origin] {
		return true
	}
	return false
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}
