package gateway

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"
)

type FaultServer struct {
	Server  *httptest.Server
	Healthy atomic.Bool
}

func NewFaultServer() *FaultServer {
	f := &FaultServer{}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if !f.Healthy.Load() {
			http.Error(w, "unhealthy", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("slow response"))
	})

	mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream error", http.StatusInternalServerError)
	})

	mux.HandleFunc("/success", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	mux.HandleFunc("/large", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, string(make([]byte, 1024*1024)))
	})

	f.Healthy.Store(true)
	f.Server = httptest.NewServer(mux)

	return f
}

func (f *FaultServer) Close() {
	f.Server.Close()
}
