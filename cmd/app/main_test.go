package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

//nolint:funlen
func TestCorsMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		method            string
		wantStatus        int
		wantNextCalled    bool
		nextHandlerStatus int
	}{
		{
			name:              "GET request passes to next handler",
			method:            http.MethodGet,
			wantStatus:        http.StatusOK,
			wantNextCalled:    true,
			nextHandlerStatus: http.StatusOK,
		},
		{
			name:              "POST request passes to next handler",
			method:            http.MethodPost,
			wantStatus:        http.StatusCreated,
			wantNextCalled:    true,
			nextHandlerStatus: http.StatusCreated,
		},
		{
			name:              "DELETE request passes to next handler",
			method:            http.MethodDelete,
			wantStatus:        http.StatusNoContent,
			wantNextCalled:    true,
			nextHandlerStatus: http.StatusNoContent,
		},
		{
			name:              "OPTIONS request does not call next handler",
			method:            http.MethodOptions,
			wantStatus:        http.StatusOK,
			wantNextCalled:    false,
			nextHandlerStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nextHandlerCalled := false
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				nextHandlerCalled = true

				w.WriteHeader(tt.nextHandlerStatus)
			})

			handler := corsMiddleware(nextHandler)

			req := httptest.NewRequest(tt.method, "/", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			// Check if next handler was called
			if nextHandlerCalled != tt.wantNextCalled {
				t.Errorf("next handler called = %v, want %v", nextHandlerCalled, tt.wantNextCalled)
			}

			// Check status code
			if rec.Code != tt.wantStatus {
				t.Errorf("status code = %v, want %v", rec.Code, tt.wantStatus)
			}

			// Check CORS headers (should be set for all requests)
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
				t.Errorf("Access-Control-Allow-Origin = %v, want *", got)
			}

			if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, DELETE, OPTIONS" {
				t.Errorf("Access-Control-Allow-Methods = %v, want GET, POST, DELETE, OPTIONS", got)
			}

			expectedHeaders := "Content-Type, Authorization, Mcp-Protocol-Version, Mcp-Session-Id"
			if got := rec.Header().Get("Access-Control-Allow-Headers"); got != expectedHeaders {
				t.Errorf("Access-Control-Allow-Headers = %v, want %v", got, expectedHeaders)
			}

			if got := rec.Header().Get("Access-Control-Expose-Headers"); got != "Mcp-Session-Id" {
				t.Errorf("Access-Control-Expose-Headers = %v, want Mcp-Session-Id", got)
			}
		})
	}
}
