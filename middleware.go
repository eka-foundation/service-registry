package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

// CORS sets permissive cross-origin resource sharing rules.
func CORS() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", strings.Join([]string{
				http.MethodHead,
				http.MethodOptions,
				http.MethodGet,
			}, ", "))
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// Logger is a middleware that logs each request, along with some useful data
// about what was requested, and what the response was.
func Logger(log *log.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			sw := statusWriter{ResponseWriter: w}

			defer func() {
				log.Println(r.Method, r.URL.Path, sw.status, r.RemoteAddr, r.UserAgent())
			}()
			next.ServeHTTP(&sw, r)
		}
		return http.HandlerFunc(fn)
	}
}

// Recover is a middleware that recovers from panics that occur for a request.
func Recover() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					http.Error(w, fmt.Sprintf("[PANIC RECOVERED] %v", err), http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}
