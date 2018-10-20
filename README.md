# Kafka K8s Version Collector

Publishes available Kubernetes versions to a Kafka topic.

## Run version collector

```bash
go run main.go \
-kafka-brokers=kafka:9092 \
-kafka-topic=application-version-available \
-kafka-schema-registry-url=http://localhost:8081 \
-v=2
```
