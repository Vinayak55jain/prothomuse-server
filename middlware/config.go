package middleware

type Config struct {
    ProjectID  string
    APIKey     string
    ServerURL  string // e.g., "wss://your-server.com/stream"
    Enabled    bool
}

func DefaultConfig() *Config {
    return &Config{
        Enabled:   true,
        ServerURL: "ws://localhost:8080/stream",
    }
}