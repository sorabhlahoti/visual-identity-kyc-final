package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"visual-kyc/api/internal/config"
	"visual-kyc/api/internal/domain"
	"visual-kyc/api/internal/messaging"
	"visual-kyc/api/internal/security"
	"visual-kyc/api/internal/storage"
	"visual-kyc/api/internal/utils"
)

type AsyncService struct {
	cfg       config.Config
	metadata  storage.MetadataStore
	publisher messaging.Publisher
}

func NewAsyncService(cfg config.Config, metadata storage.MetadataStore, publisher messaging.Publisher) *AsyncService {
	return &AsyncService{cfg: cfg, metadata: metadata, publisher: publisher}
}

func (s *AsyncService) Enroll(ctx context.Context, input domain.KYCInput) (*domain.AcceptedResponse, error) {
	return s.submit(ctx, "enroll", s.cfg.KafkaTopicEnroll, input)
}

func (s *AsyncService) Verify(ctx context.Context, input domain.KYCInput) (*domain.AcceptedResponse, error) {
	return s.submit(ctx, "verify", s.cfg.KafkaTopicVerify, input)
}

func (s *AsyncService) Status(transactionID string) (*domain.StatusRecord, error) {
	return s.metadata.GetStatus(transactionID)
}

func (s *AsyncService) submit(ctx context.Context, typ, topic string, input domain.KYCInput) (*domain.AcceptedResponse, error) {
	if err := ValidateInput(input); err != nil {
		return nil, err
	}
	transactionID := utils.NewID("txn")
	job := domain.KYCJob{
		TransactionID: transactionID,
		Type:          typ,
		ImageBase64:   base64.StdEncoding.EncodeToString(input.ImageBytes),
		Name:          input.Name,
		DOB:           input.DOB,
		Gender:        input.Gender,
		CallbackURL:   input.CallbackURL,
		SubmittedAt:   time.Now().UTC(),
	}
	nonce, ciphertext, err := security.SealJSON(s.cfg.HashPepper, job)
	if err != nil {
		return nil, err
	}
	envelope := domain.EncryptedJobEnvelope{TransactionID: transactionID, Type: typ, Nonce: nonce, Ciphertext: ciphertext, CreatedAt: time.Now().UTC(), SchemaVersion: "v1"}
	if err := s.metadata.SaveStatus(domain.StatusRecord{TransactionID: transactionID, Type: typ, Status: domain.StatusAccepted, CallbackURL: input.CallbackURL, CreatedAt: time.Now().UTC()}); err != nil {
		return nil, err
	}
	log.Printf("submit kyc job type=%s topic=%s transaction_id=%s", typ, topic, transactionID)
	if err := s.publisher.Publish(topic, transactionID, "JOB_SUBMITTED", envelope); err != nil {
		_ = s.metadata.SaveStatus(domain.StatusRecord{TransactionID: transactionID, Type: typ, Status: domain.StatusFailed, Error: err.Error(), CallbackURL: input.CallbackURL, CreatedAt: time.Now().UTC()})
		return nil, fmt.Errorf("submit job to kafka failed: %w", err)
	}
	return &domain.AcceptedResponse{TransactionID: transactionID, Type: typ, KafkaTopic: topic, Status: domain.StatusAccepted, Message: "request accepted; processing is asynchronous", StatusURL: "/kyc/status/" + transactionID, AcceptedAt: time.Now().UTC()}, nil
}
