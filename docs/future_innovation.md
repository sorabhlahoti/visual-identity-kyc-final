# Further Innovation Ideas

1. Replace heuristic liveness with a trained anti-spoofing model such as Silent-Face-Anti-Spoofing or a depth/IR signal if hardware allows.
2. Add active liveness: blink, smile, random head-turn challenge, or phrase-based video challenge.
3. Add document OCR + face-document matching for stronger KYC.
4. Add W3C Verifiable Credentials so a successful KYC emits a signed credential linked to the DID.
5. Add ZK proof layer: prove a user is KYC-verified without exposing identity details.
6. Add model drift monitoring: watch score distributions and false-match review queues.
7. Add tenant isolation: per-tenant Qdrant collections, JWT scopes, and per-tenant hash peppers.
8. Add vector encryption or secure enclaves for regulated deployments.
