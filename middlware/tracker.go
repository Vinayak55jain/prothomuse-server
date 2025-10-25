package middleware

import (
    "net/http"
    "time"
)

type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}

func HealthTracker(config *Config) func(http.Handler) http.Handler {
    if !config.Enabled {
        return func(next http.Handler) http.Handler {
            return next
        }
    }
    
    client := NewClient(config)
    
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            
            // Wrap the ResponseWriter to capture status code
            wrapped := &responseWriter{
                ResponseWriter: w,
                statusCode:     http.StatusOK,
            }
            
            // Call the next handler
            next.ServeHTTP(wrapped, r)
            
            // Calculate response time
            duration := time.Since(start).Milliseconds()
            
            // Send metric to server (non-blocking)
            go func() {
                metric := MetricData{
                    Route:        r.URL.Path,
                    Method:       r.Method,
                    StatusCode:   wrapped.statusCode,
                    ResponseTime: duration,
                    Timestamp:    time.Now(),
                }
                client.SendMetric(metric)
            }()
        })
    }
}