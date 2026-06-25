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
	"strconv"

	goapitomcp "github.com/alexliesenfeld/openapimcp"
)

func main() {
	addr := envString("ADDR", ":8081")
	baseURL, err := url.Parse("http://localhost" + addr + "/api")
	if err != nil {
		log.Fatal(err)
	}

	mcpHandler, err := goapitomcp.NewHandlerFromFile(context.Background(), examplePath("openapi.yaml"), goapitomcp.Config{
		BaseURL: baseURL,
		BeforeRequest: func(_ context.Context, _ goapitomcp.OperationContext, r *http.Request) error {
			r.Header.Set("X-API-Key", "example-key")
			return nil
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/pets", handlePets)
	mux.Handle("/mcp", mcpHandler)

	log.Printf("petstore API: http://localhost%s/api/pets?limit=2", addr)
	log.Printf("MCP endpoint: http://localhost%s/mcp", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func handlePets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 || limit > len(examplePets) {
			limit = len(examplePets)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"pets":     examplePets[:limit],
			"nextPage": "",
		})
	case http.MethodPost:
		if r.Header.Get("X-API-Key") == "" {
			http.Error(w, "missing API key", http.StatusUnauthorized)
			return
		}
		var input struct {
			Name string `json:"name"`
			Tag  string `json:"tag"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if input.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"id":   len(examplePets) + 1,
			"name": input.Name,
			"tag":  input.Tag,
		})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

var examplePets = []map[string]any{
	{"id": 1, "name": "Milo", "tag": "cat"},
	{"id": 2, "name": "Otis", "tag": "dog"},
	{"id": 3, "name": "Nori", "tag": "cat"},
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
