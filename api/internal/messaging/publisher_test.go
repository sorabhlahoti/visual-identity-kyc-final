package messaging

import (
	"errors"
	"testing"
)

type fakePublisher struct {
	err    error
	called bool
}

func (f *fakePublisher) Publish(topic, transactionID, eventType string, payload interface{}) error {
	f.called = true
	return f.err
}

func TestPrimaryWithAuditPublisherIgnoresAuditFailure(t *testing.T) {
	primary := &fakePublisher{}
	audit := &fakePublisher{err: errors.New("permission denied")}
	pub := NewPrimaryWithAuditPublisher(primary, audit)
	if err := pub.Publish("topic", "txn_1", "JOB_SUBMITTED", nil); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !primary.called || !audit.called {
		t.Fatalf("expected primary and audit publishers to be called")
	}
}

func TestPrimaryWithAuditPublisherRequiresPrimarySuccess(t *testing.T) {
	primary := &fakePublisher{err: errors.New("kafka down")}
	audit := &fakePublisher{}
	pub := NewPrimaryWithAuditPublisher(primary, audit)
	if err := pub.Publish("topic", "txn_1", "JOB_SUBMITTED", nil); err == nil {
		t.Fatalf("expected primary error")
	}
	if audit.called {
		t.Fatalf("audit should not run when primary publish fails")
	}
}

func TestMultiPublisherSucceedsWhenAtLeastOnePublisherWorks(t *testing.T) {
	bad := &fakePublisher{err: errors.New("permission denied")}
	good := &fakePublisher{}
	pub := NewMultiPublisher(bad, good)
	if err := pub.Publish("topic", "txn_1", "EVENT", nil); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
