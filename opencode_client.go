package main

import (
	"context"

	opencode "github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
)

type opencodeClient struct {
	client *opencode.Client
}

func newOpencodeClient(baseURL string) ReviewClient {
	return &opencodeClient{
		client: opencode.NewClient(
			option.WithBaseURL(baseURL),
			option.WithMaxRetries(1),
		),
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
	resp, err := c.client.Session.Prompt(ctx, sessionID, opencode.SessionPromptParams{
		Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
			opencode.TextPartInputParam{
				Type: opencode.F(opencode.TextPartInputTypeText),
				Text: opencode.F(prompt),
			},
		}),
	})
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
