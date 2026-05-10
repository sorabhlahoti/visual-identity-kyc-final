# Requirement Mapping

| Requirement | Implementation |
|---|---|
| KYC enrollment API | `POST /kyc/enroll` in Go API |
| Re-KYC verification API | `POST /kyc/verify` in Go API |
| Async-first architecture | API returns `202 ACCEPTED`; worker processes via Kafka-compatible Redpanda |
| Kafka topics | `kyc_enroll`, `kyc_verify` |
| Status API | `GET /kyc/status/{transaction_id}` |
| Callback URL | Optional `callback_url` multipart field; worker POSTs result |
| 512D face embeddings | Python inference service with ArcFace ONNX; mock mode only for local smoke testing |
| 768D name embeddings | Deterministic 768D normalized name embedding |
| Separate vector collections | Qdrant `face_embeddings` and `name_embeddings` |
| Vector DB indexing optimization | Qdrant cosine vectors + HNSW config + payload index on `identity_id` |
| Secure demographic hash | HMAC-SHA256 over DOB + gender using `HASH_PEPPER` |
| Avoid raw biometric storage | No image saved to Qdrant/Redis/disk; Kafka job payload is AES-GCM encrypted and temporary |
| Liveness / anti-spoof | Python image quality + texture + face-size liveness heuristic; replaceable with trained model |
| Explainable scoring | Response includes weights and decision reasons |
| DID integration | `did:web:<issuer>:kyc:<hash>` generated per identity |
| Observability | JSON logs, `/metrics`, Prometheus, Grafana dashboard |
| Kubernetes | Helm chart in `helm/visual-kyc` |
| API docs | `api/openapi.yaml` |
| Architecture diagram | `docs/architecture.md` and `docs/diagrams/architecture.mmd` |
