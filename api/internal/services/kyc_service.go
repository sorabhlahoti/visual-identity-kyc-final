package services

import (
	"context"
	"fmt"
	"time"

	"visual-kyc/api/internal/config"
	"visual-kyc/api/internal/did"
	"visual-kyc/api/internal/domain"
	"visual-kyc/api/internal/messaging"
	"visual-kyc/api/internal/security"
	"visual-kyc/api/internal/storage"
	"visual-kyc/api/internal/utils"
)

type KYCService struct {
	cfg       config.Config
	embedder  Embedder
	vectors   storage.VectorStore
	metadata  storage.MetadataStore
	publisher messaging.Publisher
}

func NewKYCService(cfg config.Config, embedder Embedder, vectors storage.VectorStore, metadata storage.MetadataStore, publisher messaging.Publisher) *KYCService {
	return &KYCService{cfg: cfg, embedder: embedder, vectors: vectors, metadata: metadata, publisher: publisher}
}

func (s *KYCService) Enroll(ctx context.Context, input domain.KYCInput) (*domain.KYCResponse, error) {
	return s.ProcessEnroll(ctx, utils.NewID("txn"), input)
}

func (s *KYCService) Verify(ctx context.Context, input domain.KYCInput) (*domain.KYCResponse, error) {
	return s.ProcessVerify(ctx, utils.NewID("txn"), input)
}

func (s *KYCService) ProcessEnroll(ctx context.Context, transactionID string, input domain.KYCInput) (*domain.KYCResponse, error) {
	s.saveStatus(transactionID, "enroll", domain.StatusProcessing, nil, "", input.CallbackURL)
	_ = s.publisher.Publish(s.cfg.KafkaTopicEnroll, transactionID, "PROCESSING_STARTED", map[string]string{"name_hash": security.NameHash(s.cfg.HashPepper, input.Name)})

	emb, err := s.embedder.Embed(ctx, input.ImageBytes, "upload.jpg", input.Name)
	if err != nil {
		s.saveStatus(transactionID, "enroll", domain.StatusFailed, nil, err.Error(), input.CallbackURL)
		return nil, err
	}

	demoHash := security.DemographicHash(s.cfg.HashPepper, input.DOB, input.Gender)
	nameHash := security.NameHash(s.cfg.HashPepper, input.Name)

	faceMatches, err := s.vectors.SearchFace(ctx, emb.FaceEmbedding, s.cfg.TopK)
	if err != nil {
		s.saveStatus(transactionID, "enroll", domain.StatusFailed, nil, err.Error(), input.CallbackURL)
		return nil, err
	}
	nameMatches, err := s.vectors.SearchName(ctx, emb.NameEmbedding, s.cfg.TopK)
	if err != nil {
		s.saveStatus(transactionID, "enroll", domain.StatusFailed, nil, err.Error(), input.CallbackURL)
		return nil, err
	}

	best := s.bestCandidate(faceMatches, nameMatches, demoHash)
	live := toDomainLiveness(emb.Liveness)
	if best.IdentityID != "" && best.FaceScore >= s.cfg.FaceDupThreshold {
		responseStatus := domain.StatusAlreadyExists
		if !best.DemographicMatch && best.NameScore >= s.cfg.NameMatchThreshold {
			responseStatus = domain.StatusPotentialFraud
		}
		if best.NameScore < s.cfg.NameMatchThreshold && best.DemographicMatch {
			responseStatus = domain.StatusPartialMatch
		}
		identityDID := ""
		if meta, err := s.metadata.GetIdentity(best.IdentityID); err == nil {
			identityDID = meta.DID
		}
		resp := &domain.KYCResponse{
			TransactionID:   transactionID,
			IdentityID:      best.IdentityID,
			DID:             identityDID,
			Status:          responseStatus,
			ConfidenceScore: computeConfidence(best.FaceScore, best.NameScore, best.DemographicMatch),
			Details: domain.MatchDetails{
				CandidateIdentityID: best.IdentityID,
				FaceSimilarity:      round4(best.FaceScore),
				NameSimilarity:      round4(best.NameScore),
				DemographicMatch:    best.DemographicMatch,
				Liveness:            &live,
				Explainability:      explainDecision(responseStatus, best.FaceScore, best.NameScore, best.DemographicMatch, live.Passed),
			},
		}
		s.saveStatus(transactionID, "enroll", domain.StatusCompleted, resp, "", input.CallbackURL)
		_ = s.publisher.Publish(s.cfg.KafkaTopicEnroll, transactionID, "DUPLICATE_DECISION", resp)
		return resp, nil
	}

	identityID := utils.NewUUID()
	identityDID := did.ForIdentity(s.cfg.DIDIssuer, identityID)
	if err := s.vectors.UpsertFace(ctx, identityID, identityID, emb.FaceEmbedding); err != nil {
		s.saveStatus(transactionID, "enroll", domain.StatusFailed, nil, err.Error(), input.CallbackURL)
		return nil, err
	}
	if err := s.vectors.UpsertName(ctx, identityID, identityID, emb.NameEmbedding); err != nil {
		s.saveStatus(transactionID, "enroll", domain.StatusFailed, nil, err.Error(), input.CallbackURL)
		return nil, err
	}
	if err := s.metadata.SaveIdentity(domain.IdentityMetadata{
		IdentityID:      identityID,
		DID:             identityDID,
		DemographicHash: demoHash,
		NameHash:        nameHash,
		CreatedAt:       time.Now().UTC(),
		Status:          "ACTIVE",
	}); err != nil {
		s.saveStatus(transactionID, "enroll", domain.StatusFailed, nil, err.Error(), input.CallbackURL)
		return nil, err
	}

	resp := &domain.KYCResponse{
		TransactionID:   transactionID,
		IdentityID:      identityID,
		DID:             identityDID,
		Status:          domain.StatusNewUser,
		ConfidenceScore: 1.0,
		Details: domain.MatchDetails{
			FaceSimilarity:   round4(best.FaceScore),
			NameSimilarity:   round4(best.NameScore),
			DemographicMatch: best.DemographicMatch,
			Liveness:         &live,
			Explainability:   explainDecision(domain.StatusNewUser, best.FaceScore, best.NameScore, best.DemographicMatch, live.Passed),
		},
	}
	s.saveStatus(transactionID, "enroll", domain.StatusCompleted, resp, "", input.CallbackURL)
	_ = s.publisher.Publish(s.cfg.KafkaTopicEnroll, transactionID, "NEW_IDENTITY_CREATED", resp)
	return resp, nil
}

func (s *KYCService) ProcessVerify(ctx context.Context, transactionID string, input domain.KYCInput) (*domain.KYCResponse, error) {
	s.saveStatus(transactionID, "verify", domain.StatusProcessing, nil, "", input.CallbackURL)
	_ = s.publisher.Publish(s.cfg.KafkaTopicVerify, transactionID, "PROCESSING_STARTED", map[string]string{"name_hash": security.NameHash(s.cfg.HashPepper, input.Name)})

	emb, err := s.embedder.Embed(ctx, input.ImageBytes, "upload.jpg", input.Name)
	if err != nil {
		s.saveStatus(transactionID, "verify", domain.StatusFailed, nil, err.Error(), input.CallbackURL)
		return nil, err
	}

	demoHash := security.DemographicHash(s.cfg.HashPepper, input.DOB, input.Gender)
	faceMatches, err := s.vectors.SearchFace(ctx, emb.FaceEmbedding, s.cfg.TopK)
	if err != nil {
		s.saveStatus(transactionID, "verify", domain.StatusFailed, nil, err.Error(), input.CallbackURL)
		return nil, err
	}
	nameMatches, err := s.vectors.SearchName(ctx, emb.NameEmbedding, s.cfg.TopK)
	if err != nil {
		s.saveStatus(transactionID, "verify", domain.StatusFailed, nil, err.Error(), input.CallbackURL)
		return nil, err
	}

	best := s.bestCandidate(faceMatches, nameMatches, demoHash)
	status := domain.StatusNoMatch
	live := toDomainLiveness(emb.Liveness)
	if best.IdentityID != "" && live.Passed && best.FaceScore >= s.cfg.FaceMatchThreshold && best.NameScore >= s.cfg.NameMatchThreshold && best.DemographicMatch {
		status = domain.StatusMatched
	} else if best.IdentityID != "" && live.Passed && best.FaceScore >= s.cfg.PartialFaceThreshold && (best.NameScore >= 0.55 || best.DemographicMatch) {
		status = domain.StatusPartialMatch
	}
	identityDID := ""
	if best.IdentityID != "" {
		if meta, err := s.metadata.GetIdentity(best.IdentityID); err == nil {
			identityDID = meta.DID
		}
	}
	resp := &domain.KYCResponse{
		TransactionID:   transactionID,
		IdentityID:      best.IdentityID,
		DID:             identityDID,
		Status:          status,
		ConfidenceScore: computeConfidence(best.FaceScore, best.NameScore, best.DemographicMatch),
		Details: domain.MatchDetails{
			CandidateIdentityID: best.IdentityID,
			FaceSimilarity:      round4(best.FaceScore),
			NameSimilarity:      round4(best.NameScore),
			DemographicMatch:    best.DemographicMatch,
			Liveness:            &live,
			Explainability:      explainDecision(status, best.FaceScore, best.NameScore, best.DemographicMatch, live.Passed),
		},
	}
	s.saveStatus(transactionID, "verify", domain.StatusCompleted, resp, "", input.CallbackURL)
	_ = s.publisher.Publish(s.cfg.KafkaTopicVerify, transactionID, "VERIFY_DECISION", resp)
	return resp, nil
}

func (s *KYCService) Status(transactionID string) (*domain.StatusRecord, error) {
	return s.metadata.GetStatus(transactionID)
}

type candidateScore struct {
	IdentityID       string
	FaceScore        float64
	NameScore        float64
	DemographicMatch bool
}

func (s *KYCService) bestCandidate(faceMatches, nameMatches []storage.SearchResult, demoHash string) candidateScore {
	nameScores := map[string]float64{}
	for _, n := range nameMatches {
		if n.IdentityID != "" && n.Score > nameScores[n.IdentityID] {
			nameScores[n.IdentityID] = n.Score
		}
	}

	best := candidateScore{}
	bestCombined := -1.0
	for _, f := range faceMatches {
		if f.IdentityID == "" {
			continue
		}
		meta, err := s.metadata.GetIdentity(f.IdentityID)
		if err != nil {
			continue
		}
		nameScore := nameScores[f.IdentityID]
		demoMatch := meta.DemographicHash == demoHash
		combined := computeConfidence(f.Score, nameScore, demoMatch)
		if combined > bestCombined {
			bestCombined = combined
			best = candidateScore{IdentityID: f.IdentityID, FaceScore: f.Score, NameScore: nameScore, DemographicMatch: demoMatch}
		}
	}
	return best
}

func computeConfidence(face, name float64, demo bool) float64 {
	demoScore := 0.0
	if demo {
		demoScore = 1.0
	}
	return round4(utils.Clamp01(0.65*face + 0.25*name + 0.10*demoScore))
}

func round4(v float64) float64 {
	return float64(int(v*10000+0.5)) / 10000
}

func (s *KYCService) saveStatus(transactionID, typ, status string, result *domain.KYCResponse, errMsg, callbackURL string) {
	_ = s.metadata.SaveStatus(domain.StatusRecord{
		TransactionID: transactionID,
		Type:          typ,
		Status:        status,
		Result:        result,
		Error:         errMsg,
		CallbackURL:   callbackURL,
		CreatedAt:     time.Now().UTC(),
	})
}

func ValidateInput(input domain.KYCInput) error {
	if len(input.ImageBytes) == 0 {
		return fmt.Errorf("image is required")
	}
	if input.Name == "" {
		return fmt.Errorf("name is required")
	}
	if input.DOB == "" {
		return fmt.Errorf("dob is required")
	}
	if input.Gender == "" {
		return fmt.Errorf("gender is required")
	}
	return nil
}

func toDomainLiveness(l Liveness) domain.LivenessDetails {
	return domain.LivenessDetails{Passed: l.Passed, Score: round4(l.Score), Reason: l.Reason, AntiSpoofing: l.AntiSpoofing}
}

func explainDecision(status string, face, name float64, demo bool, live bool) domain.ExplainabilityInfo {
	reasons := []string{}
	if live {
		reasons = append(reasons, "liveness_passed")
	} else {
		reasons = append(reasons, "liveness_failed_or_not_available")
	}
	if face >= 0.80 {
		reasons = append(reasons, "strong_face_match")
	} else if face >= 0.72 {
		reasons = append(reasons, "moderate_face_match")
	} else {
		reasons = append(reasons, "weak_face_match")
	}
	if name >= 0.70 {
		reasons = append(reasons, "name_match")
	} else {
		reasons = append(reasons, "name_mismatch_or_not_in_shortlist")
	}
	if demo {
		reasons = append(reasons, "demographic_hash_exact_match")
	} else {
		reasons = append(reasons, "demographic_hash_mismatch")
	}
	reasons = append(reasons, "final_status_"+status)
	return domain.ExplainabilityInfo{FaceWeight: 0.65, NameWeight: 0.25, DemographicWeight: 0.10, DecisionReasons: reasons}
}
