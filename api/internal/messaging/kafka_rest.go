package messaging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type KafkaRESTPublisher struct {
	baseURL string
	client  *http.Client
	retries int
}

func NewKafkaRESTPublisher(baseURL string) *KafkaRESTPublisher {
	return NewKafkaRESTPublisherWithOptions(baseURL, 30*time.Second, 6)
}

func NewKafkaRESTPublisherWithOptions(baseURL string, timeout time.Duration, retries int) *KafkaRESTPublisher {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if retries < 1 {
		retries = 1
	}
	return &KafkaRESTPublisher{baseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{Timeout: timeout}, retries: retries}
}

func (p *KafkaRESTPublisher) Publish(topic, transactionID, eventType string, payload interface{}) error {
	evt := Event{Topic: topic, TransactionID: transactionID, Type: eventType, Payload: payload, CreatedAt: time.Now().UTC()}
	body := map[string]interface{}{"records": []map[string]interface{}{{"key": transactionID, "value": evt}}}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	var lastErr error
	for attempt := 1; attempt <= p.retries; attempt++ {
		status, respBody, err := p.do(context.Background(), http.MethodPost, p.baseURL+"/topics/"+topic, bytes.NewReader(b))
		if err == nil && status >= 200 && status < 300 {
			if err := validateProduceResponse(topic, respBody); err != nil {
				lastErr = err
			} else {
				return nil
			}
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("kafka rest publish failed topic=%s status=%d body=%s", topic, status, respBody)
			if status >= 400 && status < 500 {
				break
			}
		}
		time.Sleep(time.Duration(attempt) * 750 * time.Millisecond)
	}
	return lastErr
}

func (p *KafkaRESTPublisher) Health(ctx context.Context) error {
	status, body, err := p.do(ctx, http.MethodGet, p.baseURL+"/topics", nil)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("kafka rest health failed status=%d body=%s", status, body)
	}
	return nil
}

func (p *KafkaRESTPublisher) do(ctx context.Context, method, url string, body io.Reader) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return 0, "", err
	}
	if body != nil {
		// Redpanda HTTP Proxy requires the JSON records media type for produce requests.
		req.Header.Set("Content-Type", "application/vnd.kafka.json.v2+json")
	}
	// Produce and topic-list responses use the generic Kafka REST response type.
	// Using application/vnd.kafka.json.v2+json as Accept can return HTTP 406 not_acceptable.
	req.Header.Set("Accept", "application/vnd.kafka.v2+json")
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(respBody), nil
}

func validateProduceResponse(topic, body string) error {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return nil
	}
	var resp struct {
		Offsets []struct {
			Partition int    `json:"partition"`
			Offset    int64  `json:"offset"`
			ErrorCode int    `json:"error_code"`
			Error     string `json:"error"`
		} `json:"offsets"`
	}
	if err := json.Unmarshal([]byte(trimmed), &resp); err != nil {
		// Some Redpanda versions return a different success payload. Do not fail
		// only because the success payload is not recognized.
		return nil
	}
	if len(resp.Offsets) == 0 {
		return fmt.Errorf("kafka rest publish topic=%s returned success without offsets body=%s", topic, trimmed)
	}
	for _, off := range resp.Offsets {
		if off.ErrorCode != 0 || off.Error != "" {
			return fmt.Errorf("kafka rest publish topic=%s record error_code=%d error=%s body=%s", topic, off.ErrorCode, off.Error, trimmed)
		}
	}
	return nil
}

func WaitForKafkaREST(ctx context.Context, baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	publisher := NewKafkaRESTPublisherWithOptions(baseURL, 5*time.Second, 1)
	var last error
	for time.Now().Before(deadline) {
		if err := publisher.Health(ctx); err != nil {
			last = err
			time.Sleep(time.Second)
			continue
		}
		return nil
	}
	if last == nil {
		last = context.DeadlineExceeded
	}
	return last
}

type KafkaRESTConsumer struct {
	baseURL       string
	group         string
	name          string
	instanceURL   string
	client        *http.Client
	pollTimeoutMS int
	maxRecords    int
}

func NewKafkaRESTConsumer(baseURL, group, name string, pollTimeoutMS, maxRecords int) *KafkaRESTConsumer {
	return &KafkaRESTConsumer{
		baseURL:       strings.TrimRight(baseURL, "/"),
		group:         group,
		name:          name,
		client:        &http.Client{Timeout: 30 * time.Second},
		pollTimeoutMS: pollTimeoutMS,
		maxRecords:    maxRecords,
	}
}

func (c *KafkaRESTConsumer) Start(ctx context.Context, topics []string) error {
	createBody := map[string]interface{}{"name": c.name, "format": "json", "auto.offset.reset": "earliest"}
	status, body, err := c.do(ctx, http.MethodPost, c.baseURL+"/consumers/"+c.group, createBody)
	if err != nil {
		return err
	}
	if status == http.StatusConflict {
		_ = c.Close(context.Background())
		status, body, err = c.do(ctx, http.MethodPost, c.baseURL+"/consumers/"+c.group, createBody)
		if err != nil {
			return err
		}
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("create kafka consumer failed status=%d body=%s", status, body)
	}
	// Some proxies return an advertised base_uri with an internal Docker hostname.
	// Build the instance URL from the client configured baseURL so Windows/local clients work.
	c.instanceURL = c.baseURL + "/consumers/" + c.group + "/instances/" + c.name
	status, body, err = c.do(ctx, http.MethodPost, c.instanceURL+"/subscription", map[string]interface{}{"topics": topics})
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("subscribe kafka consumer failed status=%d body=%s", status, body)
	}
	return nil
}

func (c *KafkaRESTConsumer) Poll(ctx context.Context) ([]Event, error) {
	if c.instanceURL == "" {
		return nil, fmt.Errorf("consumer not started")
	}
	url := fmt.Sprintf("%s/records?timeout=%d&max_bytes=10485760", c.instanceURL, c.pollTimeoutMS)
	status, body, err := c.do(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNoContent {
		return nil, nil
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("poll kafka records failed status=%d body=%s", status, body)
	}
	var records []struct {
		Value Event `json:"value"`
	}
	if err := json.Unmarshal([]byte(body), &records); err != nil {
		return nil, err
	}
	events := make([]Event, 0, len(records))
	for _, r := range records {
		if r.Value.TransactionID != "" {
			events = append(events, r.Value)
		}
	}
	return events, nil
}

func (c *KafkaRESTConsumer) Close(ctx context.Context) error {
	url := c.baseURL + "/consumers/" + c.group + "/instances/" + c.name
	status, body, err := c.do(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound || status == http.StatusNoContent {
		return nil
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("delete kafka consumer failed status=%d body=%s", status, body)
	}
	return nil
}

func (c *KafkaRESTConsumer) do(ctx context.Context, method, url string, payload interface{}) (int, string, error) {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return 0, "", err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return 0, "", err
	}

	// Consumer management endpoints use application/vnd.kafka.v2+json.
	// Records polling must request JSON records explicitly.
	if strings.Contains(url, "/records") {
		req.Header.Set("Accept", "application/vnd.kafka.json.v2+json")
	} else {
		req.Header.Set("Accept", "application/vnd.kafka.v2+json")
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/vnd.kafka.v2+json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b), nil
}
