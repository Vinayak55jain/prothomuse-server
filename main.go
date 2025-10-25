package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Metric struct {
	ProjectID    string `json:"projectId"`
	Route        string `json:"route"`
	Method       string `json:"method"`
	StatusCode   int    `json:"statusCode"`
	ResponseTime int64  `json:"responseTime"`
	Timestamp    string `json:"timestamp"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Thread-safe metrics storage
var (
	metrics []Metric
	mu      sync.RWMutex
	maxMetrics = 1000 // Limit stored metrics
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := gin.Default()
	
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})
	
	r.GET("/health", func(c *gin.Context) {
		mu.RLock()
		count := len(metrics)
		mu.RUnlock()
		
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "prothomuse-health-server",
			"metrics_count": count,
		})
	})
	
	r.GET("/stream", func(c *gin.Context) {
		handleWebSocket(ctx, c)
	})
	
	r.GET("/metrics", func(c *gin.Context) {
		mu.RLock()
		defer mu.RUnlock()
		
		c.JSON(200, gin.H{
			"count":   len(metrics),
			"metrics": metrics,
		})
	})
	
	log.Println("🚀 Server starting on http://localhost:8080")
	log.Println("📊 WebSocket: ws://localhost:8080/stream")
	log.Println("🔍 Health: http://localhost:8080/health")
	log.Println("📈 Metrics: http://localhost:8080/metrics")
	
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}
	
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}

func handleWebSocket(ctx context.Context, c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()
	
	log.Println("✅ New WebSocket connection established")
	
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("WebSocket read error: %v", err)
				return
			}
			
			var metric Metric
			if err := json.Unmarshal(message, &metric); err != nil {
				log.Printf("Failed to parse metric: %v", err)
				continue
			}
			
			mu.Lock()
			if len(metrics) >= maxMetrics {
				metrics = metrics[1:] // Remove oldest metric
			}
			metrics = append(metrics, metric)
			mu.Unlock()
			
			log.Printf("📊 Received metric: %s %s -> %d (%dms)",
				metric.Method,
				metric.Route,
				metric.StatusCode,
				metric.ResponseTime,
			)
			
			ack := map[string]string{
				"status": "received",
				"route":  metric.Route,
			}
			ackJSON, err := json.Marshal(ack)
			if err != nil {
				log.Printf("Failed to marshal ack: %v", err)
				continue
			}
			
			if err := conn.WriteMessage(websocket.TextMessage, ackJSON); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}
		}
	}
}