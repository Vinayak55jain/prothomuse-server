package middleware

import (
    "encoding/json"
    "log"
    
    "github.com/gorilla/websocket"
)

type MetricData struct {
    ProjectID    string    `json:"projectId"`
    Route        string    `json:"route"`
    Method       string    `json:"method"`
    StatusCode   int       `json:"statusCode"`
    ResponseTime int64     `json:"responseTime"` // milliseconds
    Timestamp    int64 `json:"timestamp"`
}

type Client struct {
    config *Config
    conn   *websocket.Conn
}

func NewClient(config *Config) *Client {
    client := &Client{config: config}
    client.connect()
    return client
}

func (c *Client) connect() {
    var err error
    c.conn, _, err = websocket.DefaultDialer.Dial(c.config.ServerURL, nil)
    if err != nil {
        log.Printf("WebSocket connection failed: %v", err)
        return
    }
    log.Println("Connected to health tracking server")
}

func (c *Client) SendMetric(data MetricData) {
    if c.conn == nil {
        return
    }
    
    data.ProjectID = c.config.ProjectID
    
    jsonData, err := json.Marshal(data)
    if err != nil {
        log.Printf("Failed to marshal metric: %v", err)
        return
    }
    
    err = c.conn.WriteMessage(websocket.TextMessage, jsonData)
    if err != nil {
        log.Printf("Failed to send metric: %v", err)
        // Try to reconnect
        c.connect()
    }
}

func (c *Client) Close() {
    if c.conn != nil {
        c.conn.Close()
    }
}
