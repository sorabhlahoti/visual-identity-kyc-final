package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"visual-kyc/api/internal/domain"
)

type MetadataStore interface {
	SaveIdentity(meta domain.IdentityMetadata) error
	GetIdentity(identityID string) (*domain.IdentityMetadata, error)
	SaveStatus(record domain.StatusRecord) error
	GetStatus(transactionID string) (*domain.StatusRecord, error)
}

type JSONMetadataStore struct {
	path string
	mu   sync.Mutex
	data storeFile
}

type storeFile struct {
	Identities map[string]domain.IdentityMetadata `json:"identities"`
	Statuses   map[string]domain.StatusRecord     `json:"statuses"`
}

func NewJSONMetadataStore(path string) (*JSONMetadataStore, error) {
	s := &JSONMetadataStore{path: path, data: storeFile{Identities: map[string]domain.IdentityMetadata{}, Statuses: map[string]domain.StatusRecord{}}}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, s.flushLocked()
	}
	if err != nil {
		return nil, err
	}
	if len(b) > 0 {
		if err := json.Unmarshal(b, &s.data); err != nil {
			return nil, fmt.Errorf("parse metadata file: %w", err)
		}
	}
	if s.data.Identities == nil {
		s.data.Identities = map[string]domain.IdentityMetadata{}
	}
	if s.data.Statuses == nil {
		s.data.Statuses = map[string]domain.StatusRecord{}
	}
	return s, nil
}

func (s *JSONMetadataStore) SaveIdentity(meta domain.IdentityMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.loadLocked()
	now := time.Now().UTC()
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = now
	}
	meta.UpdatedAt = now
	s.data.Identities[meta.IdentityID] = meta
	return s.flushLocked()
}

func (s *JSONMetadataStore) GetIdentity(identityID string) (*domain.IdentityMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.loadLocked()
	meta, ok := s.data.Identities[identityID]
	if !ok {
		return nil, os.ErrNotExist
	}
	return &meta, nil
}

func (s *JSONMetadataStore) SaveStatus(record domain.StatusRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.loadLocked()
	now := time.Now().UTC()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now
	s.data.Statuses[record.TransactionID] = record
	return s.flushLocked()
}

func (s *JSONMetadataStore) GetStatus(transactionID string) (*domain.StatusRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.loadLocked()
	rec, ok := s.data.Statuses[transactionID]
	if !ok {
		return nil, os.ErrNotExist
	}
	return &rec, nil
}

func (s *JSONMetadataStore) loadLocked() error {
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(b) == 0 {
		return nil
	}
	var data storeFile
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	if data.Identities == nil {
		data.Identities = map[string]domain.IdentityMetadata{}
	}
	if data.Statuses == nil {
		data.Statuses = map[string]domain.StatusRecord{}
	}
	s.data = data
	return nil
}

func (s *JSONMetadataStore) flushLocked() error {
	tmp := s.path + ".tmp"
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
