package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

type EmbeddingResult struct {
	FaceEmbedding []float64 `json:"face_embedding"`
	NameEmbedding []float64 `json:"name_embedding"`
	FaceDim       int       `json:"face_dim"`
	NameDim       int       `json:"name_dim"`
	ModelInfo     string    `json:"model_info"`
	Liveness      Liveness  `json:"liveness"`
}

type Liveness struct {
	Passed       bool    `json:"passed"`
	Score        float64 `json:"score"`
	Reason       string  `json:"reason"`
	AntiSpoofing string  `json:"anti_spoofing"`
}

type Embedder interface {
	Embed(ctx context.Context, imageBytes []byte, filename, name string) (*EmbeddingResult, error)
}

type HTTPEmbedderClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPEmbedderClient(baseURL string) *HTTPEmbedderClient {
	return &HTTPEmbedderClient{baseURL: baseURL, client: &http.Client{Timeout: 90 * time.Second}}
}

func (c *HTTPEmbedderClient) Embed(ctx context.Context, imageBytes []byte, filename, name string) (*EmbeddingResult, error) {
	var lastErr error
	for attempt := 1; attempt <= 5; attempt++ {
		result, err := c.embedOnce(ctx, imageBytes, filename, name)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			break
		}
		time.Sleep(time.Duration(attempt) * 2 * time.Second)
	}
	return nil, fmt.Errorf("embedder unavailable after retries: %w", lastErr)
}

func (c *HTTPEmbedderClient) embedOnce(ctx context.Context, imageBytes []byte, filename, name string) (*EmbeddingResult, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	fw, err := writer.CreateFormFile("image", filename)
	if err != nil {
		return nil, err
	}
	if _, err := fw.Write(imageBytes); err != nil {
		return nil, err
	}
	if err := writer.WriteField("name", name); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/embed", &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embedder failed status=%d body=%s", resp.StatusCode, string(b))
	}
	var result EmbeddingResult
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, err
	}
	if result.FaceDim != 512 || len(result.FaceEmbedding) != 512 {
		return nil, fmt.Errorf("face embedding must be 512D, got dim=%d len=%d", result.FaceDim, len(result.FaceEmbedding))
	}
	if result.NameDim != 768 || len(result.NameEmbedding) != 768 {
		return nil, fmt.Errorf("name embedding must be 768D, got dim=%d len=%d", result.NameDim, len(result.NameEmbedding))
	}
	return &result, nil
}
