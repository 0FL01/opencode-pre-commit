package main

import (
	"context"
	"strings"

	opencode "github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
)

type opencodeClient struct {
	client *opencode.Client
	model  string
}

func newOpencodeClient(baseURL string) ReviewClient {
	return newOpencodeClientWithModel(baseURL, "")
}

func newOpencodeClientWithModel(baseURL, model string) ReviewClient {
	return &opencodeClient{
		client: opencode.NewClient(
			option.WithBaseURL(baseURL),
			option.WithMaxRetries(1),
		),
		model: model,
	}
}

func (c *opencodeClient) NewSession(ctx context.Context, title string) (string, error) {
	session, err := c.client.Session.New(ctx, opencode.SessionNewParams{
		Title: opencode.F(title),
	})
	if err != nil {
		return "", err
	}
	return session.ID, nil
}

func (c *opencodeClient) Prompt(ctx context.Context, sessionID string, prompt string) (string, error) {
	params := opencode.SessionPromptParams{
		Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
			opencode.TextPartInputParam{
				Type: opencode.F(opencode.TextPartInputTypeText),
				Text: opencode.F(prompt),
			},
		}),
	}

	// If model is specified, add it to the request
	if c.model != "" {
		modelID, providerID := parseModel(c.model)
		params.Model = opencode.F(opencode.SessionPromptParamsModel{
			ModelID:    opencode.F(modelID),
			ProviderID: opencode.F(providerID),
		})
	}

	resp, err := c.client.Session.Prompt(ctx, sessionID, params)
	if err != nil {
		return "", err
	}

	var text string
	for _, part := range resp.Parts {
		if tp, ok := part.AsUnion().(opencode.TextPart); ok {
			text += tp.Text
		}
	}
	return text, nil
}

func (c *opencodeClient) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := c.client.Session.Delete(ctx, sessionID, opencode.SessionDeleteParams{})
	return err
}

// parseModel splits "provider/model" into (modelID, providerID).
// If format is invalid, returns the whole string as modelID with empty providerID.
func parseModel(model string) (modelID, providerID string) {
	parts := strings.Split(model, "/")
	if len(parts) == 2 {
		return parts[1], parts[0]
	}
	return model, ""
}
