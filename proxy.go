package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/EsanSamuel/reverse-proxy/controllers"
	"github.com/EsanSamuel/reverse-proxy/db"
	"github.com/EsanSamuel/reverse-proxy/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var userBackends = make(map[string]*url.URL)

var rrIndex uint32
var healthyBackends []string // Track all healthy backend urls
var currentBackends []string // Track current project's backend urls
var mu sync.RWMutex

type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
	wroteHeader  bool
}

func NewResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		wroteHeader:    false,
	}
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	if rw.wroteHeader {
		return
	}
	rw.statusCode = statusCode
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	if err != nil {
		return 0, err
	}
	rw.bytesWritten += int64(n)
	return n, nil
}

func main() {
	go healthCheckRoutine()

	http.HandleFunc("/", proxyHandler)
	http.HandleFunc("/register", registerUserBackend)
	http.HandleFunc("/project", controllers.CreateProject)
	http.HandleFunc("/api_key", controllers.ProxyApiKey)
	http.HandleFunc("/projects", controllers.GetProxyProjects)
	fmt.Println("Proxy server is running at port 9000")
	if err := http.ListenAndServe(":9000", nil); err != nil {
		fmt.Println("Proxy server failed to connect ", err)
	}
}

func getNextBackendUrl() string {
	mu.RLock()
	defer mu.RUnlock()

	n := len(healthyBackends)
	if n == 0 {
		return ""
	}

	idx := atomic.AddUint32(&rrIndex, 1)
	target := healthyBackends[int(idx-1)%n]

	return target
}

func registerUserBackend(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	backendUrl := r.URL.Query().Get("url")

	if name == "" || backendUrl == "" {
		http.Error(w, "No name or url found", http.StatusInternalServerError)
		return
	}

	parsedUrl, err := url.Parse(backendUrl)
	fmt.Println(parsedUrl)

	if err != nil {
		log.Println("Error parsing url", err)
	}

	mu.Lock()
	userBackends[name] = parsedUrl
	mu.Unlock()
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	api_key := r.Header.Get("X-API-KEY")

	if api_key == "" {
		http.Error(w, "Missing X-API-KEY header", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var project models.Project

	err := db.Proxy_ProjectCollection.FindOne(ctx, bson.M{"api_key": api_key}).Decode(&project)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Invalid API-KEY", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Error getting project", http.StatusInternalServerError)
		return
	}

	// Update current backends for health checking
	mu.Lock()
	currentBackends = project.BackendUrls
	// Initialize healthyBackends
	if len(healthyBackends) == 0 {
		healthyBackends = append([]string(nil), project.BackendUrls...)
	}
	mu.Unlock()

	backendUrl := getNextBackendUrl()

	if backendUrl == "" {
		http.Error(w, "No healthy backends available", http.StatusServiceUnavailable)
		return
	}

	backendURL, err := url.Parse(backendUrl)
	if err != nil {
		http.Error(w, "Bad backend URL", http.StatusInternalServerError)
		return
	}

	log.Printf("Proxying request %s %s -> %s", r.Method, r.URL.Path, backendURL)

	proxy := httputil.NewSingleHostReverseProxy(backendURL)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = backendURL.Host
		req.Header.Set("X-Forwarded-Proto", backendURL.Scheme)
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Real-IP", req.RemoteAddr)
	}

	start := time.Now()
	rw := NewResponseWriter(w)

	proxy.ServeHTTP(rw, r)

	duration := time.Since(start)

	var response_log models.ResponseLog

	response_log.ResponseLogId = bson.NewObjectID().Hex()
	response_log.UserID = project.UserID
	response_log.BytesWritten = rw.bytesWritten
	response_log.Method = r.Method
	response_log.Host = r.Host
	response_log.UrlPath = r.URL.Path
	response_log.StatusCode = rw.statusCode
	response_log.Duration = duration.Milliseconds()
	response_log.ClientIP = r.RemoteAddr
	response_log.UserAgent = r.UserAgent()
	response_log.QueryParams = r.URL.RawQuery
	response_log.Referer = r.Header.Get("Referer")
	response_log.Timestamp = time.Now()
	response_log.Protocol = r.Proto                            // HTTP/1.1, HTTP/2, etc.
	response_log.ContentType = rw.Header().Get("Content-Type") // Response type

	result, err := db.Response_Log.InsertOne(ctx, response_log)
	if result.Acknowledged {
		log.Println(result)
	}

	if err != nil {
		http.Error(w, "Error inserting response log to db", http.StatusInternalServerError)
		return
	}

	log.Printf(
		"host=%s method=%s path=%s status=%d bytes=%d duration_ms=%dms",
		r.Host,
		r.Method,
		r.URL.Path,
		rw.statusCode,
		rw.bytesWritten,
		duration.Milliseconds(),
	)
}

func healthCheckRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mu.RLock()
		backends := append([]string(nil), currentBackends...)
		mu.RUnlock()

		if len(backends) == 0 {
			continue
		}

		newHealthy := []string{}

		for _, backend := range backends {
			if isBackendHealthy(backend) {
				newHealthy = append(newHealthy, backend)
			}
		}

		mu.Lock()
		healthyBackends = newHealthy
		mu.Unlock()

		log.Printf("Healthy backends: %v", newHealthy)
	}
}

func isBackendHealthy(backend string) bool {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(backend)
	if err != nil {
		log.Printf("Backend down: %s (%v)", backend, err)
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}
