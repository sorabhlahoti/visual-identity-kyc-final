package main

import (
	"log"
	"net/http"
	"time"

	"visual-kyc/api/internal/config"
	"visual-kyc/api/internal/httpserver"
	"visual-kyc/api/internal/logging"
	"visual-kyc/api/internal/messaging"
	"visual-kyc/api/internal/metrics"
	"visual-kyc/api/internal/services"
	"visual-kyc/api/internal/storage"
)

func main() {
	cfg := config.Load()
	logger := logging.New(cfg.ServiceName)
	var metadata storage.MetadataStore
	if cfg.RedisAddr != "" {
		metadata = storage.NewRedisMetadataStore(cfg.RedisAddr, cfg.RedisPassword)
	} else {
		jsonStore, err := storage.NewJSONMetadataStore(cfg.MetadataPath)
		if err != nil {
			log.Fatalf("metadata store: %v", err)
		}
		metadata = jsonStore
	}
	// The async API must not block startup on Qdrant or Kafka.
	// Qdrant is required by the worker, not by /health or by accepting HTTP requests.
	// Kafka readiness is handled during publish with retries, so failures become JSON
	// responses instead of a dead container / empty TCP reply on Windows.
	filePub := messaging.NewFilePublisher(cfg.EventLogPath)
	kafkaPub := messaging.NewKafkaRESTPublisherWithOptions(cfg.KafkaRESTURL, time.Duration(cfg.KafkaPublishTimeoutSec)*time.Second, cfg.KafkaPublishRetries)
	publisher := messaging.NewPrimaryWithAuditPublisher(kafkaPub, filePub)
	svc := services.NewAsyncService(cfg, metadata, publisher)
	counters := &metrics.Counters{}
	server := &http.Server{Addr: ":" + cfg.Port, Handler: httpserver.NewRouter(cfg, svc, counters), ReadHeaderTimeout: time.Duration(cfg.ReadHeaderTimeoutSec) * time.Second}
	logger.Info("api server starting", map[string]interface{}{"port": cfg.Port, "kafka_rest_url": cfg.KafkaRESTURL})
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("api server stopped", map[string]interface{}{"error": err.Error()})
		log.Fatal(err)
	}
}

func retryWithLog(logger *logging.Logger, name string, attempts int, delay time.Duration, fn func() error) error {
	var last error
	for i := 1; i <= attempts; i++ {
		if err := fn(); err != nil {
			last = err
			if i == 1 || i%10 == 0 || i == attempts {
				logger.Error("dependency not ready", map[string]interface{}{"dependency": name, "attempt": i, "max_attempts": attempts, "error": err.Error()})
			}
			time.Sleep(delay)
			continue
		}
		if i > 1 {
			logger.Info("dependency ready", map[string]interface{}{"dependency": name, "attempt": i})
		}
		return nil
	}
	return last
}
