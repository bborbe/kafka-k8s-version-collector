# Send App Versions to Kafka

## Setup Kafka

```bash
echo "127.0.0.1 zookeeper" >> /etc/hosts
echo "127.0.0.1 kafka" >> /etc/hosts
echo "127.0.0.1 schema-registry" >> /etc/hosts
```

```bash
docker network create kafka
```
 
Start Zookeeper 
 
```bash
docker kill zookeeper
docker rm zookeeper
docker run \
--net=kafka \
--name=zookeeper \
-e ZOOKEEPER_CLIENT_PORT=2181 \
confluentinc/cp-zookeeper:5.0.0
```

Start Kafka
 
```bash
docker kill kafka
docker rm kafka
docker run \
--net=kafka \
--name=kafka \
-p 9092:9092 \
-e KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181 \
-e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://kafka:9092 \
-e KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1 \
confluentinc/cp-kafka:5.0.0
```

Start Schema-Registry

```bash
docker kill schema-registry
docker rm schema-registry
docker run \
--net=kafka \
--name=schema-registry \
-p 8081:8081 \
-e SCHEMA_REGISTRY_KAFKASTORE_CONNECTION_URL=zookeeper:2181 \
-e SCHEMA_REGISTRY_HOST_NAME=schema-registry \
-e SCHEMA_REGISTRY_LISTENERS=http://0.0.0.0:8081 \
confluentinc/cp-schema-registry:5.0.0
```


Consume topic on console 

```bash
docker run -ti \
--net=kafka \
confluentinc/cp-schema-registry:5.0.0 \
kafka-avro-console-consumer \
--bootstrap-server kafka:9092 \
--topic versions \
--property schema.registry.url=http://schema-registry:8081
```


## Run version collector

```bash
go run main.go \
-kafka-brokers=kafka:9092 \
-kafka-topic=versions \
-kafka-schema-registry-url=http://localhost:8081 \
-v=2
```

## Create and configure topic

```bash
kafka-topics --create --topic versions --zookeeper zookeeper:2181 --replication-factor 1 --partitions 1
kafka-configs --zookeeper zookeeper:2181  --entity-type topics --entity-name versions --alter --add-config retention.ms=-1
kafka-configs --zookeeper zookeeper:2181  --entity-type topics --entity-name versions --alter --add-config retention.bytes=-1
kafka-configs --zookeeper zookeeper:2181  --entity-type topics --entity-name versions --alter --add-config cleanup.policy=compact
```
