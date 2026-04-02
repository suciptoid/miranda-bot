package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckBanned(t *testing.T) {
	tests := []struct {
		name     string
		userID   int64
		response string
		expected bool
	}{
		{
			name:     "User is banned",
			userID:   12345,
			response: `{"ok": true}`,
			expected: true,
		},
		{
			name:     "User is not banned",
			userID:   67890,
			response: `{"ok": false}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := fmt.Sprintf("/?user_id=%d", tt.userID)
				if r.URL.String() != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.String())
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			oldURL := casBaseURL
			casBaseURL = server.URL
			defer func() { casBaseURL = oldURL }()

			result := checkBanned(tt.userID)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
