package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

// corsMiddleware adds CORS headers to all responses.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8100")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// NewRouter sets up API routes with company-scoped endpoints.
func NewRouter(handlers *Handlers, wsHandler *WebSocketHandler) *mux.Router {
	r := mux.NewRouter()

	// Apply CORS middleware
	r.Use(corsMiddleware)

	// Company management
	r.HandleFunc("/api/companies", handlers.CreateCompany).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/companies", handlers.ListCompanies).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/companies/{id}", handlers.GetCompany).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/companies/{id}", handlers.DeleteCompany).Methods("DELETE", "OPTIONS")

	// Company-scoped sessions
	r.HandleFunc("/api/companies/{id}/sessions", handlers.CreateSession).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/companies/{id}/sessions", handlers.ListSessions).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/companies/{id}/sessions/{sid}", handlers.GetSession).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/companies/{id}/sessions/{sid}/workflow", handlers.GetWorkflow).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/companies/{id}/sessions/{sid}/review", handlers.GetReview).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/companies/{id}/sessions/{sid}/approve", handlers.ApproveWorkflow).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/companies/{id}/sessions/{sid}/start", handlers.StartWorkflow).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/companies/{id}/sessions/{sid}/resume", handlers.ResumeWorkflow).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/companies/{id}/sessions/{sid}/decision", handlers.SubmitDecision).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/companies/{id}/sessions/{sid}/execute/{stepId}", handlers.ExecuteStep).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/companies/{id}/sessions/{sid}/steps/{stepId}/restart", handlers.RestartStep).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/companies/{id}/sessions/{sid}/steps/{stepId}/history", handlers.GetStepHistory).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/companies/{id}/sessions/{sid}/outputs", handlers.ListSessionOutputs).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/companies/{id}/sessions/{sid}/outputs/{filename}", handlers.GetSessionOutput).Methods("GET", "OPTIONS")

	// Session final outputs (最终产物)
	r.HandleFunc("/api/companies/{id}/sessions/{sid}/finaloutputs", handlers.ListSessionFinalOutputs).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/companies/{id}/sessions/{sid}/finaloutputs/{filename}", handlers.GetSessionFinalOutput).Methods("GET", "OPTIONS")

	// Company roles
	r.HandleFunc("/api/companies/{id}/roles", handlers.ListCompanyRoles).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/companies/{id}/roles/{roleId}", handlers.GetRole).Methods("GET", "OPTIONS")

	// WebSocket
	r.HandleFunc("/ws", wsHandler.Handle)

	return r
}