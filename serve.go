package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

//go:embed web/*
var webFS embed.FS

type serverConfig struct {
	Addr     string
	DataDir  string
	TmplPath string
	Backends map[string]Backend
}

func runServer(cfg serverConfig) error {
	mux := http.NewServeMux()

	staticFS, err := fs.Sub(webFS, "web")
	if err != nil {
		return err
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	// GET /api/sample?locale=ko  →  data/{locale}.json
	mux.HandleFunc("/api/sample", func(w http.ResponseWriter, r *http.Request) {
		locale := r.URL.Query().Get("locale")
		if locale == "" {
			http.Error(w, "missing locale", http.StatusBadRequest)
			return
		}
		path := filepath.Join(cfg.DataDir, locale+".json")
		b, err := readFile(path)
		if err != nil {
			http.Error(w, "sample not found: "+locale, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(b)
	})

	// POST /api/render  body: JSON  →  rendered HTML
	mux.HandleFunc("/api/render", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
			return
		}
		var inv Invoice
		if err := json.Unmarshal(body, &inv); err != nil {
			http.Error(w, "json parse: "+err.Error(), http.StatusBadRequest)
			return
		}
		str, ok := localeStrings[inv.Locale]
		if !ok {
			http.Error(w, "unsupported locale: "+inv.Locale, http.StatusBadRequest)
			return
		}
		inv.Strings = str
		// 합계 재계산
		var subtotal float64
		for _, it := range inv.Items {
			subtotal += it.Amount
		}
		inv.Subtotal = subtotal
		inv.Tax = roundTax(subtotal, inv.TaxRate)
		inv.Total = inv.Subtotal + inv.Tax

		html, err := renderHTML(cfg.TmplPath, &inv)
		if err != nil {
			http.Error(w, "render: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	})

	// POST /api/convert?backend=X  body: HTML  →  PDF
	mux.HandleFunc("/api/convert", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		name := r.URL.Query().Get("backend")
		b, ok := cfg.Backends[name]
		if !ok {
			http.Error(w, "unknown backend: "+name, http.StatusBadRequest)
			return
		}
		html, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
		defer cancel()

		start := time.Now()
		pdf, err := b.Convert(ctx, string(html))
		elapsed := time.Since(start)
		if err != nil {
			log.Printf("[serve/%s] FAIL %s: %v", name, elapsed.Round(time.Millisecond), err)
			http.Error(w, fmt.Sprintf("%s convert failed: %v", name, err), http.StatusBadGateway)
			return
		}
		log.Printf("[serve/%s] OK %s (%d KB)", name, elapsed.Round(time.Millisecond), len(pdf)/1024)
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `inline; filename="`+name+`.pdf"`)
		w.Header().Set("X-Convert-Backend", name)
		w.Header().Set("X-Convert-Elapsed-Ms", fmt.Sprintf("%d", elapsed.Milliseconds()))
		w.Write(pdf)
	})

	// 헬스
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":       true,
			"backends": keysOf(cfg.Backends),
		})
	})

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           logging(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("listening on http://localhost%s", cfg.Addr)
	log.Printf("→ 데모 페이지: http://localhost%s/", cfg.Addr)
	return srv.ListenAndServe()
}

func roundTax(subtotal, ratePct float64) float64 {
	return float64(int64(subtotal*ratePct+0.5)) / 100
}

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func logging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		h.ServeHTTP(sw, r)
		log.Printf("%s %s → %d (%s)", r.Method, r.URL.Path, sw.status, time.Since(start).Round(time.Millisecond))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func keysOf[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
