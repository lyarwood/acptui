// Mock API server for demo recording.
// Serves canned responses for all acptui endpoints.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/me", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"authenticated": true,
			"userId":        "demo-user",
			"username":      "demo-user",
			"displayName":   "Demo User",
		})
	})

	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"name": "my-team", "displayName": "My Team", "status": "Active"},
				{"name": "platform", "displayName": "Platform Engineering", "status": "Active"},
				{"name": "demos", "displayName": "Demo Project", "status": "Active"},
			},
		})
	})

	mux.HandleFunc("/api/projects/my-team/agentic-sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			json.NewEncoder(w).Encode(map[string]any{
				"name":    "session-new-001",
				"message": "Agentic session created successfully",
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				session("session-001", "Refactor auth middleware", "Running", "busy",
					"claude-sonnet-4-6", "https://github.com/acme/backend", "5m"),
				session("session-002", "Fix CI pipeline failures", "Running", "idle",
					"claude-sonnet-4-6", "https://github.com/acme/infra", "15m"),
				session("session-003", "Add OpenAPI docs for v2 endpoints", "Stopped", "",
					"claude-opus-4-6", "https://github.com/acme/backend", "2h"),
				session("session-004", "Review PR #342 - rate limiting", "Completed", "",
					"claude-sonnet-4-6", "https://github.com/acme/backend", "1d"),
				session("session-005", "Investigate OOM in staging pods", "Stopped", "",
					"claude-sonnet-4-5", "https://github.com/acme/infra", "3d"),
			},
		})
	})

	mux.HandleFunc("/api/projects/my-team/agentic-sessions/session-001/agui/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, _ := w.(http.Flusher)

		events := []string{
			sseSnapshot([]msg{
				{ID: "m1", Role: "user", Content: "Can you refactor the auth middleware to use the new JWT library? The current implementation has a token refresh race condition."},
				{ID: "m2", Role: "reasoning", Content: "The user wants to refactor the auth middleware. Let me analyze the current implementation to understand the race condition in token refresh. I should look at how tokens are being refreshed concurrently and identify the critical section that needs synchronization."},
				{ID: "m3", Role: "assistant", Content: "I've analyzed the auth middleware and found the race condition. Here's my plan:\n\n1. **Root cause**: `refreshToken()` in `middleware/auth.go:47` reads and writes the cached token without synchronization. Two concurrent requests can both see an expired token and trigger parallel refreshes.\n\n2. **Fix**: Replace the bare `sync.Mutex` with a `singleflight.Group` to deduplicate concurrent refresh calls.\n\n3. **Migration**: Swap `golang-jwt/jwt/v4` for `golang-jwt/jwt/v5` — the new `ParseWithClaims` API handles clock skew better.\n\nI'll start with the fix now."},
				{ID: "m4", Role: "user", Content: "Looks good, go ahead with the fix."},
				{ID: "m5", Role: "reasoning", Content: "The user approved the plan. I'll implement the singleflight-based token refresh, update the JWT library, and add proper test coverage for the concurrent refresh scenario."},
				{ID: "m6", Role: "assistant", Content: "Done! Here's what I changed:\n\n- `middleware/auth.go`: Replaced `sync.Mutex` with `singleflight.Group` for token refresh deduplication\n- `go.mod`: Updated `golang-jwt/jwt` v4 → v5\n- `middleware/auth_test.go`: Added `TestConcurrentTokenRefresh` with 50 parallel goroutines\n\nAll tests pass. The race condition is fixed — concurrent requests now share a single refresh call instead of racing."},
			}),
		}

		for _, evt := range events {
			fmt.Fprint(w, evt)
			flusher.Flush()
		}

		// Keep connection open
		<-r.Context().Done()
	})

	mux.HandleFunc("/api/projects/my-team/agentic-sessions/session-001/agui/run", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"runId":    "run-demo-001",
			"threadId": "session-001",
		})
	})

	mux.HandleFunc("/api/projects/my-team/agentic-sessions/session-001/agui/tasks", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"tasks": []map[string]any{
				{"task_id": "t1", "description": "Analyze auth middleware", "status": "completed",
					"summary": "Identified race condition in token refresh", "last_tool_name": "Read",
					"usage": map[string]any{"total_tokens": 45200, "tool_uses": 12, "duration_ms": 8500}},
				{"task_id": "t2", "description": "Apply singleflight fix", "status": "completed",
					"summary": "Refactored token refresh with singleflight.Group", "last_tool_name": "Edit",
					"usage": map[string]any{"total_tokens": 32100, "tool_uses": 8, "duration_ms": 5200}},
			},
		})
	})

	mux.HandleFunc("/api/projects/my-team/agentic-sessions/session-001/workspace", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		switch path {
		case "/repos", "":
			json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"name": "repos", "path": "/repos", "isDir": true, "size": 4096},
					{"name": "artifacts", "path": "/artifacts", "isDir": true, "size": 4096},
				},
			})
		case "/repos/backend":
			json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"name": "middleware", "path": "/repos/backend/middleware", "isDir": true, "size": 4096},
					{"name": "go.mod", "path": "/repos/backend/go.mod", "isDir": false, "size": 1240},
					{"name": "README.md", "path": "/repos/backend/README.md", "isDir": false, "size": 3400},
				},
			})
		default:
			json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"name": "backend", "path": "/repos/backend", "isDir": true, "size": 4096},
				},
			})
		}
	})

	mux.HandleFunc("/api/projects/my-team/models", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]any{
				{"id": "claude-haiku-4-5", "label": "Claude Haiku 4.5", "isDefault": false},
				{"id": "claude-sonnet-4-6", "label": "Claude Sonnet 4.6", "isDefault": true},
				{"id": "claude-opus-4-6", "label": "Claude Opus 4.6", "isDefault": false},
			},
		})
	})

	mux.HandleFunc("/api/workflows/ootb", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"workflows": []map[string]any{
				{"id": "bugfix", "name": "Fix a bug", "description": "Systematic workflow for analyzing and fixing bugs", "gitUrl": "https://github.com/ambient-code/workflows.git", "branch": "main", "path": "workflows/bugfix", "enabled": true},
				{"id": "dev-team", "name": "Dev Team", "description": "Multi-agent development team workflow", "gitUrl": "https://github.com/ambient-code/workflows.git", "branch": "main", "path": "workflows/dev-team", "enabled": true},
				{"id": "spec-kit", "name": "Spec Kit", "description": "Generate specifications from requirements", "gitUrl": "https://github.com/ambient-code/workflows.git", "branch": "main", "path": "workflows/spec-kit", "enabled": true},
			},
		})
	})

	// Catch-all for start/stop/delete
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/start") || strings.Contains(r.URL.Path, "/stop") {
			w.WriteHeader(202)
			json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
			return
		}
		if r.Method == "DELETE" {
			w.WriteHeader(204)
			return
		}
		w.WriteHeader(404)
	})

	log.Println("Mock API server listening on :9999")
	log.Fatal(http.ListenAndServe(":9999", mux))
}

func session(id, name, phase, agent, model, repo, ago string) map[string]any {
	t := time.Now()
	return map[string]any{
		"apiVersion": "vteam.ambient-code/v1alpha1",
		"kind":       "AgenticSession",
		"metadata":   map[string]any{"name": id, "namespace": "my-team", "creationTimestamp": t.Add(-1 * time.Hour).Format(time.RFC3339)},
		"spec": map[string]any{
			"displayName":   name,
			"initialPrompt": name,
			"llmSettings":   map[string]any{"model": model, "temperature": 0.7},
			"repos":         []map[string]any{{"url": repo, "branch": "main"}},
			"userContext":   map[string]any{"userId": "demo-user", "displayName": "Demo User"},
		},
		"status": map[string]any{
			"phase":       phase,
			"agentStatus": agent,
			"startTime":   t.Add(-1 * time.Hour).Format(time.RFC3339),
			"conditions": []map[string]any{
				{"type": "Ready", "status": "True", "reason": "Running", "message": "Session is running"},
				{"type": "ReposReconciled", "status": "True", "reason": "Reconciled", "message": "1 repo cloned"},
			},
			"reconciledRepos": []map[string]any{
				{"name": strings.Split(strings.TrimPrefix(repo, "https://github.com/"), "/")[1], "url": repo, "branch": "main", "status": "Ready"},
			},
		},
	}
}

type msg struct {
	ID, Role, Content string
}

func sseSnapshot(msgs []msg) string {
	var messages []map[string]any
	for _, m := range msgs {
		entry := map[string]any{"id": m.ID, "role": m.Role, "content": m.Content}
		if m.Role == "reasoning" {
			entry["role"] = "reasoning"
		}
		messages = append(messages, entry)
	}
	data, _ := json.Marshal(map[string]any{
		"type":     "MESSAGES_SNAPSHOT",
		"messages": messages,
		"runId":    "run-demo",
		"threadId": "session-001",
	})
	return fmt.Sprintf("data: %s\n\n", data)
}
