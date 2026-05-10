package messaging

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Event struct {
	Topic         string      `json:"topic"`
	TransactionID string      `json:"transaction_id"`
	Type          string      `json:"type"`
	Payload       interface{} `json:"payload,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
}

type Publisher interface {
	Publish(topic, transactionID, eventType string, payload interface{}) error
}

type FilePublisher struct {
	path string
	mu   sync.Mutex
}

func NewFilePublisher(path string) *FilePublisher {
	if path == "" {
		path = "./data/events.log"
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	return &FilePublisher{path: path}
}

func NewPublisher(path string) *FilePublisher { return NewFilePublisher(path) }

func (p *FilePublisher) Publish(topic, transactionID, eventType string, payload interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(p.path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(p.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	evt := Event{Topic: topic, TransactionID: transactionID, Type: eventType, Payload: payload, CreatedAt: time.Now().UTC()}
	b, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	return err
}

// MultiPublisher publishes to every configured publisher and succeeds when at least
// one publisher succeeds. It is useful for best-effort audit/event mirrors.
// For the main async API flow, use PrimaryWithAuditPublisher so Kafka remains mandatory.
type MultiPublisher struct {
	publishers []Publisher
}

func NewMultiPublisher(publishers ...Publisher) *MultiPublisher {
	return &MultiPublisher{publishers: publishers}
}

func (m *MultiPublisher) Publish(topic, transactionID, eventType string, payload interface{}) error {
	var errs []string
	var successes int
	for _, p := range m.publishers {
		if p == nil {
			continue
		}
		if err := p.Publish(topic, transactionID, eventType, payload); err != nil {
			errs = append(errs, err.Error())
		} else {
			successes++
		}
	}
	if successes > 0 {
		return nil
	}
	if len(errs) == 0 {
		return errors.New("no publishers configured")
	}
	return fmt.Errorf("all publishers failed: %s", strings.Join(errs, "; "))
}

// PrimaryWithAuditPublisher requires the primary publisher to succeed and writes
// audit copies through secondary publishers on a best-effort basis. This prevents
// a local audit file permission issue from rejecting a Kafka-submitted KYC job.
type PrimaryWithAuditPublisher struct {
	primary Publisher
	audits  []Publisher
}

func NewPrimaryWithAuditPublisher(primary Publisher, audits ...Publisher) *PrimaryWithAuditPublisher {
	return &PrimaryWithAuditPublisher{primary: primary, audits: audits}
}

func (p *PrimaryWithAuditPublisher) Publish(topic, transactionID, eventType string, payload interface{}) error {
	if p.primary == nil {
		return errors.New("primary publisher is not configured")
	}
	if err := p.primary.Publish(topic, transactionID, eventType, payload); err != nil {
		return err
	}
	for _, audit := range p.audits {
		if audit != nil {
			_ = audit.Publish(topic, transactionID, eventType, payload)
		}
	}
	return nil
}
