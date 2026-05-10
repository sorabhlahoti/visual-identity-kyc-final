package messaging

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestKafkaRESTPublisherRetriesAndSucceeds(t *testing.T) {
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if atomic.LoadInt32(&calls) == 1 {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"offsets":[{"partition":0,"offset":1}]}`))
	}))
	defer ts.Close()

	pub := NewKafkaRESTPublisherWithOptions(ts.URL, time.Second, 2)
	if err := pub.Publish("kyc_enroll", "txn_1", "JOB_SUBMITTED", map[string]string{"ok": "yes"}); err != nil {
		t.Fatalf("expected publish success after retry, got %v", err)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestKafkaRESTPublisherDoesNotRetryBadRequest(t *testing.T) {
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer ts.Close()

	pub := NewKafkaRESTPublisherWithOptions(ts.URL, time.Second, 5)
	if err := pub.Publish("kyc_enroll", "txn_1", "JOB_SUBMITTED", nil); err == nil {
		t.Fatalf("expected bad request error")
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected no retry for 400, got %d calls", calls)
	}
}
