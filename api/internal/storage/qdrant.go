package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	FaceVectorSize = 512
	NameVectorSize = 768
)

type VectorStore interface {
	EnsureCollections(ctx context.Context) error
	UpsertFace(ctx context.Context, pointID, identityID string, vector []float64) error
	UpsertName(ctx context.Context, pointID, identityID string, vector []float64) error
	SearchFace(ctx context.Context, vector []float64, limit int) ([]SearchResult, error)
	SearchName(ctx context.Context, vector []float64, limit int) ([]SearchResult, error)
}

type QdrantClient struct {
	baseURL        string
	faceCollection string
	nameCollection string
	httpClient     *http.Client
}

type SearchResult struct {
	ID         string                 `json:"id"`
	Score      float64                `json:"score"`
	Payload    map[string]interface{} `json:"payload"`
	IdentityID string                 `json:"-"`
}

func NewQdrantClient(baseURL, faceCollection, nameCollection string) *QdrantClient {
	return &QdrantClient{
		baseURL:        baseURL,
		faceCollection: faceCollection,
		nameCollection: nameCollection,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (q *QdrantClient) EnsureCollections(ctx context.Context) error {
	if err := q.ensureCollection(ctx, q.faceCollection, FaceVectorSize); err != nil {
		return err
	}
	if err := q.ensureCollection(ctx, q.nameCollection, NameVectorSize); err != nil {
		return err
	}
	_ = q.ensurePayloadIndex(ctx, q.faceCollection, "identity_id")
	_ = q.ensurePayloadIndex(ctx, q.nameCollection, "identity_id")
	return nil
}

func (q *QdrantClient) ensureCollection(ctx context.Context, name string, size int) error {
	payload := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     size,
			"distance": "Cosine",
		},
		"hnsw_config": map[string]interface{}{
			"m":            16,
			"ef_construct": 100,
		},
	}
	status, body, err := q.doJSON(ctx, http.MethodPut, fmt.Sprintf("/collections/%s", name), payload)
	if err != nil {
		return err
	}
	if status == http.StatusConflict {
		// Qdrant returns 409 when the collection already exists.
		// This is safe and expected when the API is restarted against an existing Qdrant volume.
		return nil
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("qdrant create collection %s failed status=%d body=%s", name, status, body)
	}
	return nil
}

func (q *QdrantClient) ensurePayloadIndex(ctx context.Context, collection, field string) error {
	payload := map[string]interface{}{
		"field_name":   field,
		"field_schema": "keyword",
	}
	status, body, err := q.doJSON(ctx, http.MethodPut, fmt.Sprintf("/collections/%s/index", collection), payload)
	if err != nil {
		return err
	}
	if status == http.StatusConflict || status == http.StatusBadRequest {
		return nil
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("qdrant create payload index %s.%s failed status=%d body=%s", collection, field, status, body)
	}
	return nil
}

func (q *QdrantClient) UpsertFace(ctx context.Context, pointID, identityID string, vector []float64) error {
	return q.upsert(ctx, q.faceCollection, pointID, identityID, vector)
}

func (q *QdrantClient) UpsertName(ctx context.Context, pointID, identityID string, vector []float64) error {
	return q.upsert(ctx, q.nameCollection, pointID, identityID, vector)
}

func (q *QdrantClient) upsert(ctx context.Context, collection, pointID, identityID string, vector []float64) error {
	payload := map[string]interface{}{
		"points": []map[string]interface{}{
			{
				"id":     pointID,
				"vector": vector,
				"payload": map[string]interface{}{
					"identity_id": identityID,
				},
			},
		},
	}
	status, body, err := q.doJSON(ctx, http.MethodPut, fmt.Sprintf("/collections/%s/points?wait=true", collection), payload)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("qdrant upsert %s failed status=%d body=%s", collection, status, body)
	}
	return nil
}

func (q *QdrantClient) SearchFace(ctx context.Context, vector []float64, limit int) ([]SearchResult, error) {
	return q.search(ctx, q.faceCollection, vector, limit)
}

func (q *QdrantClient) SearchName(ctx context.Context, vector []float64, limit int) ([]SearchResult, error) {
	return q.search(ctx, q.nameCollection, vector, limit)
}

func (q *QdrantClient) search(ctx context.Context, collection string, vector []float64, limit int) ([]SearchResult, error) {
	payload := map[string]interface{}{
		"vector":       vector,
		"limit":        limit,
		"with_payload": true,
	}
	status, body, err := q.doJSON(ctx, http.MethodPost, fmt.Sprintf("/collections/%s/points/search", collection), payload)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("qdrant search %s failed status=%d body=%s", collection, status, body)
	}
	var parsed struct {
		Result []SearchResult `json:"result"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return nil, err
	}
	for i := range parsed.Result {
		if id, ok := parsed.Result[i].Payload["identity_id"].(string); ok {
			parsed.Result[i].IdentityID = id
		}
	}
	return parsed.Result, nil
}

func (q *QdrantClient) doJSON(ctx context.Context, method, path string, payload interface{}) (int, string, error) {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return 0, "", err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, q.baseURL+path, body)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := q.httpClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b), nil
}
