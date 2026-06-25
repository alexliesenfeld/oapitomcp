package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	goapitomcp "github.com/alexliesenfeld/openapimcp"
)

func main() {
	addr := envString("ADDR", ":8083")
	baseURL, err := url.Parse("http://localhost" + addr + "/api")
	if err != nil {
		log.Fatal(err)
	}

	mcpHandler, err := goapitomcp.NewHandlerFromFile(context.Background(), examplePath("openapi.yaml"), goapitomcp.Config{
		BaseURL: baseURL,
		Filter: &goapitomcp.OperationFilter{
			ExcludePathPatterns: []string{"/admin/*"},
			ExcludeTags:         []string{"debug", "internal"},
			ExcludeMethods:      []string{http.MethodDelete},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/users", handleUsers)
	mux.HandleFunc("/api/admin/users", handleAdminUsers)
	mux.HandleFunc("/api/debug/status", handleDebugStatus)
	mux.Handle("/mcp", mcpHandler)

	log.Printf("filtered API: http://localhost%s/api/users", addr)
	log.Printf("MCP endpoint: http://localhost%s/mcp", addr)
	log.Println("exposed MCP tools: listUsers, createUser")
	log.Println("filtered out: deleteAllUsers, debugStatus")
	log.Fatal(http.ListenAndServe(addr, mux))
}

func handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"users": []map[string]any{
				{"id": "usr_1", "email": "alex@example.test"},
				{"id": "usr_2", "email": "sam@example.test"},
			},
		})
	case http.MethodPost:
		var input struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if input.Email == "" {
			http.Error(w, "email is required", http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"id":    "usr_new",
			"email": input.Email,
		})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleDebugStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func examplePath(name string) string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("cannot locate example directory")
	}
	return filepath.Join(filepath.Dir(file), name)
}

func envString(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
