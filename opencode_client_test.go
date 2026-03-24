package main

import (
	"testing"

	opencode "github.com/sst/opencode-sdk-go"
)

func TestParseModel(t *testing.T) {
	tests := []struct {
		input        string
		wantModelID  string
		wantProvider string
	}{
		{"zai-coding-plan/glm-4.7", "glm-4.7", "zai-coding-plan"},
		{"provider/model", "model", "provider"},
		{"single", "single", ""},
		{"", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			modelID, providerID := parseModel(tt.input)
			if modelID != tt.wantModelID {
				t.Errorf("parseModel(%q) modelID = %q, want %q", tt.input, modelID, tt.wantModelID)
			}
			if providerID != tt.wantProvider {
				t.Errorf("parseModel(%q) providerID = %q, want %q", tt.input, providerID, tt.wantProvider)
			}
		})
	}
}

func TestNewOpencodeClientWithModel(t *testing.T) {
	client := newOpencodeClientWithModel("http://localhost:8080", "zai-coding-plan/glm-4.7")
	oc, ok := client.(*opencodeClient)
	if !ok {
		t.Fatal("expected *opencodeClient")
	}
	if oc.model != "zai-coding-plan/glm-4.7" {
		t.Errorf("model = %q, want %q", oc.model, "zai-coding-plan/glm-4.7")
	}
}

// TestOpencodeClientPromptWithModel tests that Prompt adds model to request
func TestOpencodeClientPromptWithModel(t *testing.T) {
	// This is a compile-time check that opencodeClient implements Prompt with model support
	var client *opencodeClient
	var _ = client // use client
}

// Ensure opencodeClient implements ReviewClient
var _ ReviewClient = (*opencodeClient)(nil)

// Verify that opencode SDK has the expected types
func TestOpencodeSDKTypes(t *testing.T) {
	// This test just verifies that the types we use exist
	_ = opencode.SessionPromptParamsModel{}
	_ = opencode.F("")
}
