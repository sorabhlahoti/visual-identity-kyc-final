package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port                   string
	ServiceName            string
	QdrantURL              string
	EmbedderURL            string
	MetadataPath           string
	EventLogPath           string
	FaceCollection         string
	NameCollection         string
	HashPepper             string
	JWTSecret              string
	AuthRequired           bool
	MaxImageBytes          int64
	TopK                   int
	FaceDupThreshold       float64
	FaceMatchThreshold     float64
	NameMatchThreshold     float64
	PartialFaceThreshold   float64
	KafkaRESTURL           string
	KafkaTopicEnroll       string
	KafkaTopicVerify       string
	KafkaConsumerGroup     string
	KafkaPollTimeoutMS     int
	KafkaMaxPollRecords    int
	KafkaPublishTimeoutSec int
	KafkaPublishRetries    int
	CallbackTimeoutMS      int
	ProcessingMode         string
	DIDIssuer              string
	RedisAddr              string
	RedisPassword          string
	ReadHeaderTimeoutSec   int
	CORSAllowedOrigins     string
}

func Load() Config {
	return Config{
		Port:                   getEnv("PORT", "8080"),
		ServiceName:            getEnv("SERVICE_NAME", "kyc-api"),
		QdrantURL:              strings.TrimRight(getEnv("QDRANT_URL", "http://localhost:6333"), "/"),
		EmbedderURL:            strings.TrimRight(getEnv("EMBEDDER_URL", "http://localhost:8001"), "/"),
		MetadataPath:           getEnv("METADATA_PATH", "./data/metadata.json"),
		EventLogPath:           getEnv("EVENT_LOG_PATH", "./data/events.log"),
		FaceCollection:         getEnv("FACE_COLLECTION", "face_embeddings"),
		NameCollection:         getEnv("NAME_COLLECTION", "name_embeddings"),
		HashPepper:             getEnv("HASH_PEPPER", "dev-change-this-long-random-secret-min-32-chars"),
		JWTSecret:              getEnv("JWT_SECRET", "dev-jwt-secret-change-this-min-32-chars"),
		AuthRequired:           getBool("AUTH_REQUIRED", false),
		MaxImageBytes:          getInt64("MAX_IMAGE_BYTES", 5*1024*1024),
		TopK:                   getInt("TOP_K", 8),
		FaceDupThreshold:       getFloat("FACE_DUP_THRESHOLD", 0.82),
		FaceMatchThreshold:     getFloat("FACE_MATCH_THRESHOLD", 0.80),
		NameMatchThreshold:     getFloat("NAME_MATCH_THRESHOLD", 0.70),
		PartialFaceThreshold:   getFloat("PARTIAL_FACE_THRESHOLD", 0.72),
		KafkaRESTURL:           strings.TrimRight(getEnv("KAFKA_REST_URL", "http://localhost:8082"), "/"),
		KafkaTopicEnroll:       getEnv("KAFKA_TOPIC_ENROLL", "kyc_enroll"),
		KafkaTopicVerify:       getEnv("KAFKA_TOPIC_VERIFY", "kyc_verify"),
		KafkaConsumerGroup:     getEnv("KAFKA_CONSUMER_GROUP", "kyc-workers"),
		KafkaPollTimeoutMS:     getInt("KAFKA_POLL_TIMEOUT_MS", 1000),
		KafkaMaxPollRecords:    getInt("KAFKA_MAX_POLL_RECORDS", 10),
		KafkaPublishTimeoutSec: getInt("KAFKA_PUBLISH_TIMEOUT_SEC", 30),
		KafkaPublishRetries:    getInt("KAFKA_PUBLISH_RETRIES", 6),
		CallbackTimeoutMS:      getInt("CALLBACK_TIMEOUT_MS", 2500),
		ProcessingMode:         strings.ToLower(getEnv("PROCESSING_MODE", "async")),
		DIDIssuer:              getEnv("DID_ISSUER", "did:web:localhost"),
		RedisAddr:              getEnv("REDIS_ADDR", ""),
		RedisPassword:          getEnv("REDIS_PASSWORD", ""),
		ReadHeaderTimeoutSec:   getInt("READ_HEADER_TIMEOUT_SEC", 10),
		CORSAllowedOrigins:     getEnv("CORS_ALLOWED_ORIGINS", "*"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getBool(key string, fallback bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return fallback
	}
	return parsed
}

func getInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return parsed
}

func getInt64(key string, fallback int64) int64 {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func getFloat(key string, fallback float64) float64 {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return fallback
	}
	return parsed
}
