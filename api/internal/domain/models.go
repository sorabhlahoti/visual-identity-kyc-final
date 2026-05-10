package domain

import "time"

type KYCInput struct {
	ImageBytes  []byte
	Name        string
	DOB         string
	Gender      string
	CallbackURL string
}

type IdentityMetadata struct {
	IdentityID      string    `json:"identity_id"`
	DID             string    `json:"did,omitempty"`
	DemographicHash string    `json:"demographic_hash"`
	NameHash        string    `json:"name_hash"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Status          string    `json:"status"`
}

type StatusRecord struct {
	TransactionID string       `json:"transaction_id"`
	Type          string       `json:"type"`
	Status        string       `json:"status"`
	Result        *KYCResponse `json:"result,omitempty"`
	Error         string       `json:"error,omitempty"`
	CallbackURL   string       `json:"callback_url,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

type MatchDetails struct {
	CandidateIdentityID string             `json:"candidate_identity_id,omitempty"`
	FaceSimilarity      float64            `json:"face_similarity"`
	NameSimilarity      float64            `json:"name_similarity"`
	DemographicMatch    bool               `json:"demographic_match"`
	Liveness            *LivenessDetails   `json:"liveness,omitempty"`
	Explainability      ExplainabilityInfo `json:"explainability"`
}

type LivenessDetails struct {
	Passed       bool    `json:"passed"`
	Score        float64 `json:"score"`
	Reason       string  `json:"reason,omitempty"`
	AntiSpoofing string  `json:"anti_spoofing"`
}

type ExplainabilityInfo struct {
	FaceWeight        float64  `json:"face_weight"`
	NameWeight        float64  `json:"name_weight"`
	DemographicWeight float64  `json:"demographic_weight"`
	DecisionReasons   []string `json:"decision_reasons"`
}

type KYCResponse struct {
	TransactionID   string       `json:"transaction_id"`
	IdentityID      string       `json:"identity_id,omitempty"`
	DID             string       `json:"did,omitempty"`
	Status          string       `json:"status"`
	ConfidenceScore float64      `json:"confidence_score"`
	Details         MatchDetails `json:"details"`
}

type AcceptedResponse struct {
	TransactionID string    `json:"transaction_id"`
	Type          string    `json:"type,omitempty"`
	KafkaTopic    string    `json:"kafka_topic,omitempty"`
	Status        string    `json:"status"`
	Message       string    `json:"message"`
	StatusURL     string    `json:"status_url"`
	AcceptedAt    time.Time `json:"accepted_at"`
}

type KYCJob struct {
	TransactionID string    `json:"transaction_id"`
	Type          string    `json:"type"`
	ImageBase64   string    `json:"image_base64"`
	Name          string    `json:"name"`
	DOB           string    `json:"dob"`
	Gender        string    `json:"gender"`
	CallbackURL   string    `json:"callback_url,omitempty"`
	SubmittedAt   time.Time `json:"submitted_at"`
}

type EncryptedJobEnvelope struct {
	TransactionID string    `json:"transaction_id"`
	Type          string    `json:"type"`
	Nonce         string    `json:"nonce"`
	Ciphertext    string    `json:"ciphertext"`
	CreatedAt     time.Time `json:"created_at"`
	SchemaVersion string    `json:"schema_version"`
}

const (
	StatusAccepted       = "ACCEPTED"
	StatusProcessing     = "PROCESSING"
	StatusCompleted      = "COMPLETED"
	StatusFailed         = "FAILED"
	StatusMatched        = "MATCHED"
	StatusPartialMatch   = "PARTIAL_MATCH"
	StatusNoMatch        = "NO_MATCH"
	StatusAlreadyExists  = "ALREADY_EXISTS"
	StatusNewUser        = "NEW_USER"
	StatusPotentialFraud = "POTENTIAL_FRAUD"
)
