package main

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/oneliang/company/internal/api"
	"github.com/oneliang/company/internal/company"
	"github.com/oneliang/company/internal/logging"
	"github.com/oneliang/company/internal/session"
)

func main() {
	// Initialize structured logging
	logging.Init("info", "logs/server.log")
	slog.Info("Server Starting", "log_file", "logs/server.log")

	// Base directory for company-centric storage: data/companys/<company_id>/...
	// 使用绝对路径，确保无论从哪个目录运行都能正确定位
	workDir, err := os.Getwd()
	if err != nil {
		slog.Error("Failed to get working directory", "error", err)
		os.Exit(1)
	}
	baseDir := filepath.Join(workDir, "data", "companys")
	slog.Info("Data directory", "path", baseDir)

	// Initialize stores with same baseDir
	companyStore, err := company.NewStore(baseDir)
	if err != nil {
		slog.Error("Failed to create company store", "error", err)
		os.Exit(1)
	}

	sessionStore, err := session.NewStore(baseDir)
	if err != nil {
		slog.Error("Failed to create session store", "error", err)
		os.Exit(1)
	}

	// Setup WebSocket handler for real-time progress
	wsHandler := api.NewWebSocketHandler()

	// Setup API handlers with WebSocket support
	handlers := api.NewHandlers(companyStore, sessionStore, "configs", baseDir, wsHandler)

	// Setup router
	router := api.NewRouter(handlers, wsHandler)

	// Start server
	slog.Info("Server listening", "port", 8181)
	if err := http.ListenAndServe(":8181", router); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
