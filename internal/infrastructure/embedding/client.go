// Package embedding 提供 Embedding 服务客户端
package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"z-novel-ai-api/internal/config"
)

type Client struct {
	endpoint   string
	model      string
	batchSize  int
	httpClient *http.Client
}

type embedRequest struct {
	Texts []string `json:"texts"`
	Model string   `json:"model"`
}

type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
	TokensUsed int         `json:"tokens_used"`
}

func NewClient(cfg *config.EmbeddingConfig) *Client {
	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 32
	}
	model := cfg.Model
	if model == "" {
		model = "BAAI/bge-m3"
	}
	return &Client{
		endpoint:  cfg.Endpoint,
		model:     model,
		batchSize: batchSize,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	var all [][]float32
	for i := 0; i < len(texts); i += c.batchSize {
		end := i + c.batchSize
		if end > len(texts) {
			end = len(texts)
		}

		resp, err := c.doBatchEmbed(ctx, texts[i:end])
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Embeddings...)
	}

	return all, nil
}

func (c *Client) doBatchEmbed(ctx context.Context, texts []string) (*embedResponse, error) {
	reqBody, err := json.Marshal(&embedRequest{
		Texts: texts,
		Model: c.model,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embed request: %w", err)
	}

	endpoint := strings.TrimRight(c.endpoint, "/")
	if endpoint == "" {
		return nil, fmt.Errorf("embedding endpoint is empty")
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid embedding endpoint: %w", err)
	}
	if u.Path == "" || u.Path == "/" {
		u.Path = "/embed"
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create embed request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("embedding request failed: status=%d", httpResp.StatusCode)
	}

	var resp embedResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode embed response: %w", err)
	}
	return &resp, nil
}
