package storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEnsureCollectionIgnoresAlreadyExistsConflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || !strings.HasPrefix(r.URL.Path, "/collections/") {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"status":{"error":"Wrong input: Collection already exists!"}}`))
	}))
	defer server.Close()

	client := NewQdrantClient(server.URL, "face_embeddings", "name_embeddings")
	if err := client.EnsureCollections(context.Background()); err != nil {
		t.Fatalf("expected 409 conflict to be ignored, got error: %v", err)
	}
}
