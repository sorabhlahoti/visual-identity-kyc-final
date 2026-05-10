#!/bin/sh
set -eu

echo "Waiting for Redpanda Kafka API on redpanda:9092..."
i=0
while ! rpk cluster info -X brokers=redpanda:9092 >/dev/null 2>&1; do
  i=$((i + 1))
  if [ "$i" -ge 120 ]; then
    echo "Redpanda did not become ready after 240 seconds. Last cluster info output:"
    rpk cluster info -X brokers=redpanda:9092 || true
    exit 1
  fi
  sleep 2
done

echo "Redpanda is reachable. Creating topics..."
rpk topic create kyc_enroll -X brokers=redpanda:9092 --partitions 3 --replicas 1 --if-not-exists
rpk topic create kyc_verify -X brokers=redpanda:9092 --partitions 3 --replicas 1 --if-not-exists

echo "Topics after creation:"
rpk topic list -X brokers=redpanda:9092
