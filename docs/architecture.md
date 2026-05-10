# Visual Identity KYC Final Architecture

```mermaid
flowchart TD
    C[Client / Postman / Mobile App] --> AG[API Gateway / JWT Auth]
    AG --> API[KYC API - Go]
    API -->|202 transaction_id| C
    API -->|encrypted job event| K[(Kafka-compatible Redpanda Topics: kyc_enroll / kyc_verify)]
    W[KYC Worker - Go] -->|poll jobs| K
    W --> INF[Inference Service - Python + uv: ArcFace 512D + Name 768D + Liveness]
    W --> Q[(Qdrant Vector DB: face_embeddings / name_embeddings, HNSW indexes)]
    W --> R[(Redis KV Metadata: hashes / status / DID)]
    W --> CB[Optional Callback URL]
    C -->|GET /kyc/status/{transaction_id}| API
    API --> R
    API --> M[Prometheus /metrics]
    M --> G[Grafana Dashboard]
```

## Runtime flow

1. Client sends multipart request to `POST /kyc/enroll` or `POST /kyc/verify`.
2. Go API validates request, creates `transaction_id`, encrypts the job payload using AES-GCM, stores `ACCEPTED` status, and publishes to Kafka-compatible Redpanda topic.
3. Go worker consumes the job, decrypts it in memory, calls Python inference service, receives:
   - 512D face embedding
   - 768D name embedding
   - liveness / anti-spoof score
4. Worker searches Qdrant `face_embeddings`, refines with `name_embeddings`, checks Redis demographic hash, and creates explainable decision.
5. Worker writes final status to Redis and optionally POSTs the result to `callback_url`.
6. Client polls `GET /kyc/status/{transaction_id}` for final result.

## Security model

- Raw images are never stored in Qdrant, Redis, or metadata files.
- Async Kafka payload is encrypted with AES-GCM before publishing.
- DOB + gender are HMAC-SHA256 hashed with a secret pepper.
- Names are stored only as HMAC hashes.
- JWT auth can be enabled with `AUTH_REQUIRED=true`.
