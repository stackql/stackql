package mcp_server //nolint:testpackage,revive // fine for now

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	if err := config.Validate(); err != nil {
		t.Fatalf("Default config validation failed: %v", err)
	}

	if config.Server.Name == "" {
		t.Error("Server name should not be empty")
	}

	if config.Server.Version == "" {
		t.Error("Server version should not be empty")
	}
}

// func TestConfigValidation(t *testing.T) {
// 	tests := []struct {
// 		name      string
// 		config    *Config
// 		wantError bool
// 	}{
// 		{
// 			name:      "valid default config",
// 			config:    DefaultConfig(),
// 			wantError: false,
// 		},
// 		{
// 			name: "empty server name",
// 			config: &Config{
// 				Server: ServerConfig{
// 					Name:                  "",
// 					Version:               "1.0.0",
// 					MaxConcurrentRequests: 100,
// 				},
// 				Backend: BackendConfig{
// 					Type:           "stackql",
// 					MaxConnections: 10,
// 				},
// 			},
// 			wantError: true,
// 		},
// 		{
// 			name: "invalid transport",
// 			config: &Config{
// 				Server: ServerConfig{
// 					Name:                  "Test Server",
// 					Version:               "1.0.0",
// 					MaxConcurrentRequests: 100,
// 				},
// 				Backend: BackendConfig{
// 					Type:           "stackql",
// 					MaxConnections: 10,
// 				},
// 			},
// 			wantError: true,
// 		},
// 	}
//  for _, tt := range tests {
//  	t.Run(tt.name, func(t *testing.T) {
//  		err := tt.config.Validate()
//  		if (err != nil) != tt.wantError {
//  			t.Errorf("Config.Validate() error = %v, wantError %v", err, tt.wantError)
//  		}
//  	})
//  }
// }

func TestExampleBackend(t *testing.T) {
	backend := NewExampleBackend("test://localhost")
	ctx := context.Background()

	// Test Ping
	if err := backend.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	// Test Close
	if err := backend.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestMCPServerCreation(t *testing.T) {
	config := DefaultConfig()
	backend := NewExampleBackend("test://localhost")

	server, err := newMCPServer(config, backend, nil)
	if err != nil {
		t.Fatalf("NewMCPServer failed: %v", err)
	}

	if server == nil {
		t.Fatal("Server should not be nil")
	}

	// Test that server implements MCPServer interface
	var _ MCPServer = server
}

func TestDurationMarshaling(t *testing.T) {
	d := Duration(30 * time.Second)

	// Test JSON marshaling
	jsonData, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var d2 Duration
	if err := json.Unmarshal(jsonData, &d2); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if time.Duration(d) != time.Duration(d2) {
		t.Errorf("Duration mismatch after JSON round-trip: %v != %v", d, d2)
	}
}

func TestBackendError(t *testing.T) {
	err := &BackendError{
		Code:    "TEST_ERROR",
		Message: "Test error message",
		Details: map[string]interface{}{"field": "value"},
	}

	if err.Error() != "Test error message" {
		t.Errorf("Expected error message 'Test error message', got '%s'", err.Error())
	}

	// Test Value() method for database compatibility
	val, dbErr := err.Value()
	if dbErr != nil {
		t.Fatalf("Value() failed: %v", dbErr)
	}

	if val != "Test error message" {
		t.Errorf("Expected value 'Test error message', got '%v'", val)
	}
}

func TestNewMCPServerWithExampleBackend(t *testing.T) {
	server, err := NewMCPServerWithExampleBackend(nil)
	if err != nil {
		t.Fatalf("NewMCPServerWithExampleBackend failed: %v", err)
	}

	if server == nil {
		t.Fatal("Server should not be nil")
	}
}
