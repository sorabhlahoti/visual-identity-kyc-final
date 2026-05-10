package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"visual-kyc/api/internal/callback"
	"visual-kyc/api/internal/config"
	"visual-kyc/api/internal/domain"
	"visual-kyc/api/internal/logging"
	"visual-kyc/api/internal/messaging"
	"visual-kyc/api/internal/security"
	"visual-kyc/api/internal/services"
	"visual-kyc/api/internal/storage"
	"visual-kyc/api/internal/utils"
)

func main() {
	cfg := config.Load()
	cfg.ServiceName = "kyc-worker"
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

	vectors := storage.NewQdrantClient(cfg.QdrantURL, cfg.FaceCollection, cfg.NameCollection)
	if err := retryWithLog(logger, "qdrant ensure collections", 120, 2*time.Second, func() error { return vectors.EnsureCollections(context.Background()) }); err != nil {
		log.Fatalf("qdrant ensure collections: %v", err)
	}

	embedder := services.NewHTTPEmbedderClient(cfg.EmbedderURL)
	if err := retryWithLog(logger, "kafka rest readiness", 120, 2*time.Second, func() error { return messaging.WaitForKafkaREST(context.Background(), cfg.KafkaRESTURL, 5*time.Second) }); err != nil {
		log.Fatalf("kafka rest not ready: %v", err)
	}

	publisher := messaging.NewPrimaryWithAuditPublisher(
		messaging.NewKafkaRESTPublisherWithOptions(cfg.KafkaRESTURL, time.Duration(cfg.KafkaPublishTimeoutSec)*time.Second, cfg.KafkaPublishRetries),
		messaging.NewFilePublisher(cfg.EventLogPath),
	)
	processor := services.NewKYCService(cfg, embedder, vectors, metadata, publisher)
	cb := callback.New(time.Duration(cfg.CallbackTimeoutMS) * time.Millisecond)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Use one Kafka REST consumer per command topic. This avoids a local-dev failure
	// mode where a single mixed-topic consumer group advances/blocks offsets in a
	// confusing way and verify requests remain ACCEPTED even though enroll works.
	// Each consumer group is isolated so enroll traffic cannot hide verify jobs.
	enrollConsumer := messaging.NewKafkaRESTConsumer(
		cfg.KafkaRESTURL,
		cfg.KafkaConsumerGroup+"-enroll",
		"worker-enroll-"+utils.NewID("node"),
		cfg.KafkaPollTimeoutMS,
		cfg.KafkaMaxPollRecords,
	)
	verifyConsumer := messaging.NewKafkaRESTConsumer(
		cfg.KafkaRESTURL,
		cfg.KafkaConsumerGroup+"-verify",
		"worker-verify-"+utils.NewID("node"),
		cfg.KafkaPollTimeoutMS,
		cfg.KafkaMaxPollRecords,
	)

	if err := retryWithLog(logger, "kafka enroll consumer start", 120, 2*time.Second, func() error { return enrollConsumer.Start(ctx, []string{cfg.KafkaTopicEnroll}) }); err != nil {
		log.Fatalf("kafka enroll consumer start: %v", err)
	}
	if err := retryWithLog(logger, "kafka verify consumer start", 120, 2*time.Second, func() error { return verifyConsumer.Start(ctx, []string{cfg.KafkaTopicVerify}) }); err != nil {
		log.Fatalf("kafka verify consumer start: %v", err)
	}
	defer enrollConsumer.Close(context.Background())
	defer verifyConsumer.Close(context.Background())

	logger.Info("worker started", map[string]interface{}{
		"topics": []string{cfg.KafkaTopicEnroll, cfg.KafkaTopicVerify},
		"mode":   "separate-topic-consumers",
		"groups": []string{cfg.KafkaConsumerGroup + "-enroll", cfg.KafkaConsumerGroup + "-verify"},
	})

	var wg sync.WaitGroup
	wg.Add(2)
	go consumeLoop(ctx, &wg, logger, cfg, metadata, processor, cb, enrollConsumer, cfg.KafkaTopicEnroll)
	go consumeLoop(ctx, &wg, logger, cfg, metadata, processor, cb, verifyConsumer, cfg.KafkaTopicVerify)
	wg.Wait()
}

func consumeLoop(ctx context.Context, wg *sync.WaitGroup, logger *logging.Logger, cfg config.Config, metadata storage.MetadataStore, processor *services.KYCService, cb *callback.Client, consumer *messaging.KafkaRESTConsumer, topic string) {
	defer wg.Done()
	logger.Info("consumer loop started", map[string]interface{}{"topic": topic})
	for ctx.Err() == nil {
		events, err := consumer.Poll(ctx)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				return
			}
			logger.Error("poll failed", map[string]interface{}{"topic": topic, "error": err.Error()})
			time.Sleep(time.Second)
			continue
		}
		if len(events) > 0 {
			logger.Info("events polled", map[string]interface{}{"topic": topic, "count": len(events)})
		}
		for _, evt := range events {
			if evt.Type != "JOB_SUBMITTED" {
				logger.Info("event ignored", map[string]interface{}{"topic": topic, "event_type": evt.Type, "transaction_id": evt.TransactionID})
				continue
			}
			logger.Info("job received", map[string]interface{}{"transaction_id": evt.TransactionID, "event_type": evt.Type, "topic": evt.Topic})
			if err := processEvent(ctx, cfg, metadata, processor, cb, evt); err != nil {
				logger.Error("job failed", map[string]interface{}{"transaction_id": evt.TransactionID, "topic": topic, "error": err.Error()})
			} else {
				logger.Info("job completed", map[string]interface{}{"transaction_id": evt.TransactionID, "topic": topic})
			}
		}
	}
}

func processEvent(ctx context.Context, cfg config.Config, metadata storage.MetadataStore, processor *services.KYCService, cb *callback.Client, evt messaging.Event) error {
	jobType := "unknown"
	if evt.Topic == cfg.KafkaTopicEnroll {
		jobType = "enroll"
	} else if evt.Topic == cfg.KafkaTopicVerify {
		jobType = "verify"
	}
	fail := func(err error) error {
		_ = metadata.SaveStatus(domain.StatusRecord{
			TransactionID: evt.TransactionID,
			Type:          jobType,
			Status:        domain.StatusFailed,
			Error:         err.Error(),
			CreatedAt:     time.Now().UTC(),
		})
		return err
	}

	payloadBytes, err := json.Marshal(evt.Payload)
	if err != nil {
		return fail(err)
	}
	var envelope domain.EncryptedJobEnvelope
	if err := json.Unmarshal(payloadBytes, &envelope); err != nil {
		return fail(err)
	}
	if envelope.Type != "" {
		jobType = envelope.Type
	}

	var job domain.KYCJob
	if err := security.OpenJSON(cfg.HashPepper, envelope.Nonce, envelope.Ciphertext, &job); err != nil {
		return fail(err)
	}
	if job.Type != "" {
		jobType = job.Type
	}

	imageBytes, err := base64.StdEncoding.DecodeString(job.ImageBase64)
	if err != nil {
		return fail(err)
	}
	input := domain.KYCInput{ImageBytes: imageBytes, Name: job.Name, DOB: job.DOB, Gender: job.Gender, CallbackURL: job.CallbackURL}
	var resp *domain.KYCResponse
	if job.Type == "enroll" {
		resp, err = processor.ProcessEnroll(ctx, job.TransactionID, input)
	} else if job.Type == "verify" {
		resp, err = processor.ProcessVerify(ctx, job.TransactionID, input)
	} else {
		return fail(errors.New("unknown job type: " + strings.TrimSpace(job.Type)))
	}
	if err != nil {
		return fail(err)
	}
	return cb.Post(ctx, job.CallbackURL, resp)
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
