# 🚀 gRPC MQ – Lightweight Message Queue Broker

**gRPC MQ** is a **scalable, high-performance message queue broker** built using **Go (Golang) and gRPC**. It facilitates **real-time messaging** with a **topic-based publish-subscribe (pub-sub) model**.

## 🔹 Features
✅ **Publish-Subscribe Model** – Clients subscribe to topics and receive messages in real-time.<br />
✅ **List Active Topics** – Retrieve live topics along with connected subscribers.<br />
✅ **Environment & Command-Line Configurations** – Supports `.env` file and CLI arguments.<br />
✅ **TLS Security** – Enables secure communication using SSL certificates.<br />
✅ **Logging System** – Tracks subscribers, unsubscriptions, and published messages.<br />
✅ **Docker Support** – Easily run the broker in a containerized environment.

## 📌 gRPC API Overview

### 1. Subscribe to a Topic**
Clients subscribe using `topic` and `subscriberId`, then wait for messages.
```proto
message SubscriptionRequest {
    string topic = 1;
    string subscriberId = 2;
}
```

### 2. Publish Messages
Publishers send messages to topics, and all subscribers receive them.
```proto
message Message {
    string topic = 1;
    optional string eventId = 2;
    oneof content {
        string text = 3;  // Text-based messages
        bytes binary = 4; // Binary data messages
    }
    string senderId = 5;
    int64 timestamp = 6;
}
```

### 3. List Active Topics
Retrieve live topics with connected subscribers.
```proto
message ListTopicsRequest {
    optional string topic = 1;  // If empty, lists all active topics
}
```

## ⚙️ Configuration Settings
gRPC MQ allows configuration via `.env` and command-line arguments.

### .env Configuration
```Ini
PORT=:50051
TLS_ENABLED=true  # Set to false if TLS is not needed
TLS_CERT_FILE=../certs/server.crt # Set empty if TLS is false
TLS_KEY_FILE=../certs/server.pem # Set empty if TLS is false
LOG_FILE=""  # Set log file path or leave empty
```

### Command-Line Arguments
If both `.env` and command-line arguments are provided, CLI overrides `.env`.
```Bash
go run main.go -port=:50051 -tlsEnabled=true -tlsCertFile="../certs/server.crt" -tlsKeyFile="../certs/server.pem" -logFile="../logs.log"
```

### Default Values (If Neither .env nor CLI Provided)
```Ini
PORT=:50051
TLS_ENABLED=false
TLS_CERT_FILE=""
TLS_KEY_FILE=""
LOG_FILE=""
```

## 🐳 Running with Docker
You can pass configuration using ENV variables or directly via `docker run`.
```Bash
docker run -e PORT=50051 -e TLS_ENABLED=true -e LOG_FILE=logs.log grpc-mq
```

## 📝 Logging
gRPC MQ logs all subscribers, unsubscriptions, and published records.
Users can configure logging via `.env` (`LOG_FILE`) or CLI (`-logFile`).

## 🚀 Getting Started
### 1. Clone the repository
```Bash
git clone https://github.com/yourusername/grpc-mq.git
cd grpc-mq/server
```

### 2. Build and run the project
```Bash
go run main.go
```

### 3. Subscribe and publish messages using gRPC clients!

## 📜 License
This project is licensed under the MIT License – see the `LICENSE` file for details.
