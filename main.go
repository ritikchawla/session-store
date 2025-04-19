package main

import (
	"net/http"
	"os"
	"strconv"
	"time"

	as "github.com/aerospike/aerospike-client-go/v6"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var (
	asClient          *as.Client
	sessionTTLSeconds int
)

func init() {
	host := os.Getenv("AEROSPIKE_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	port, err := strconv.Atoi(os.Getenv("AEROSPIKE_PORT"))
	if err != nil {
		port = 3000
	}
	var errClient error
	asClient, errClient = as.NewClient(host, port)
	if errClient != nil {
		panic("Failed to connect to Aerospike: " + errClient.Error())
	}

	ttlEnv := os.Getenv("SESSION_TTL")
	ttl, err := strconv.Atoi(ttlEnv)
	if err != nil || ttl <= 0 {
		ttl = 1800
	}
	sessionTTLSeconds = ttl
}

type SessionRecord struct {
	UserID    string `json:"user_id"`
	CreatedAt int64  `json:"created_at"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	Device    string `json:"device"`
	LastUsed  int64  `json:"last_used"`
	IsValid   bool   `json:"is_valid"`
}

func main() {
	router := gin.Default()

	router.POST("/session", createSession)
	router.GET("/session/:token", validateSession)
	router.DELETE("/session/:token", deleteSession)
	router.GET("/session/:token/logs", getSessionLogs)
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	router.Run() // listens on :8080 by default
}

func createSession(c *gin.Context) {
	var req struct {
		UserID    string `json:"user_id" binding:"required"`
		IP        string `json:"ip" binding:"required"`
		UserAgent string `json:"user_agent" binding:"required"`
		Device    string `json:"device"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token := uuid.NewString()
	now := time.Now().Unix()

	key, _ := as.NewKey("test", "sessions", token)
	bins := as.BinMap{
		"user_id":    req.UserID,
		"created_at": now,
		"ip":         req.IP,
		"user_agent": req.UserAgent,
		"device":     req.Device,
		"last_used":  now,
		"is_valid":   true,
	}
	asClient.Put(nil, key, bins)
	logEvent(token, req.UserID, "create", req.IP+"|"+req.UserAgent)

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func validateSession(c *gin.Context) {
	token := c.Param("token")
	key, _ := as.NewKey("test", "sessions", token)
	rec, err := asClient.Get(nil, key)
	if err != nil || rec == nil {
		c.JSON(http.StatusNotFound, gin.H{"valid": false})
		return
	}
	if valid, _ := rec.Bins["is_valid"].(bool); !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"valid": false})
		return
	}

	now := time.Now().Unix()
	asClient.Put(nil, key, as.BinMap{"last_used": now})
	userID, _ := rec.Bins["user_id"].(string)
	ip, _ := rec.Bins["ip"].(string)
	userAgent, _ := rec.Bins["user_agent"].(string)
	logEvent(token, userID, "validate", ip+"|"+userAgent)

	c.JSON(http.StatusOK, gin.H{"valid": true, "user_id": userID})
}

func deleteSession(c *gin.Context) {
	token := c.Param("token")
	key, _ := as.NewKey("test", "sessions", token)

	// Soft delete
	asClient.Put(nil, key, as.BinMap{"is_valid": false})
	logEvent(token, "", "delete", "")

	c.Status(http.StatusNoContent)
}

func getSessionLogs(c *gin.Context) {
	token := c.Param("token")
	policy := as.NewScanPolicy()
	setName := "session_logs"
	results := make([]map[string]interface{}, 0)

	if asClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Aerospike client not initialized"})
		return
	}
	it, err := asClient.ScanAll(policy, "test", setName)
	if err != nil || it == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan logs"})
		return
	}
	for res := range it.Results() {
		if res.Err != nil || res.Record == nil {
			continue
		}
		b := res.Record.Bins
		tokenVal, ok := b["token"].(string)
		if !ok || tokenVal != token {
			continue
		}
		userID, _ := b["user_id"].(string)
		createdAt, okTime := b["timestamp"].(int64)
		if !okTime {
			createdAt = 0
		}
		ip, _ := b["ip"].(string)
		userAgent, _ := b["user_agent"].(string)
		action, _ := b["action"].(string)
		results = append(results, map[string]interface{}{
			"user_id":    userID,
			"timestamp":  createdAt,
			"ip":         ip,
			"user_agent": userAgent,
			"action":     action,
		})
	}
	c.JSON(http.StatusOK, gin.H{"logs": results})
}

func logEvent(token, userID, action, ip string) {
	key, _ := as.NewKey("test", "session_logs", uuid.NewString())
	now := time.Now().Unix()
	userAgent := ""
	if action == "create" || action == "validate" {
		userAgent = ip // fallback: store IP in user_agent if not available
	}
	bins := as.BinMap{
		"token":      token,
		"user_id":    userID,
		"action":     action,
		"timestamp":  now,
		"ip":         ip,
		"user_agent": userAgent,
	}
	wp := as.NewWritePolicy(0, uint32(sessionTTLSeconds))
	if asClient == nil {
		return
	}
	_ = asClient.Put(wp, key, bins)
}
