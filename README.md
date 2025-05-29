# gRPC-mq
gRPC MQ is a lightweight, scalable message queue (MQ) broker built using Go (Golang) and gRPC, designed for efficient real-time messaging and topic-based publish-subscribe (pub-sub) communication.

# Protoc command to generate .go files
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/mq.proto
