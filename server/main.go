package main

import (
	"context"
	"log"
	"net"
	"os"
	"sync"

	pb "github.com/faizalom/grpc-mq/proto"
	"github.com/joho/godotenv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type SubscriberMessage struct {
	EventId      *string
	ContentText  string
	ContentBytes []byte
	SenderId     string
	Timestamp    int64
}

type Broker struct {
	pb.UnimplementedMessageBrokerServer
	subscribers map[string][]chan SubscriberMessage
	mu          sync.RWMutex
}

func (b *Broker) Publish(ctx context.Context, msg *pb.Message) (*pb.Response, error) {
	log.Printf("Publishing message to topic: %s, sender: %s, eventId: %v", msg.Topic, msg.GetSenderId(), msg.GetEventId())

	b.mu.RLock()
	for _, ch := range b.subscribers[msg.Topic] {
		ch <- SubscriberMessage{
			EventId:      msg.EventId,
			ContentText:  msg.GetText(),
			ContentBytes: msg.GetBinary(),
			SenderId:     msg.GetSenderId(),
			Timestamp:    msg.GetTimestamp(),
		}
	}
	b.mu.RUnlock()
	return &pb.Response{Success: true}, nil
}

func (b *Broker) Subscribe(req *pb.SubscriptionRequest, stream pb.MessageBroker_SubscribeServer) error {
	// Return an error if the topic is empty or topic length is greater than 100
	if req.Topic == "" || len(req.Topic) > 100 {
		return status.Errorf(codes.InvalidArgument, "invalid topic: %s", req.Topic)
	}

	log.Printf("New subscription request for topic: %s", req.Topic)

	ch := make(chan SubscriberMessage)
	b.mu.Lock()
	b.subscribers[req.Topic] = append(b.subscribers[req.Topic], ch)
	b.mu.Unlock()

	ctx := stream.Context()
	go func() {
		<-ctx.Done()
		log.Printf("Client disconnected from topic: %s", req.Topic)
	}()

	for msg := range ch {
		pbMess := &pb.Message{
			Topic:     req.Topic,
			EventId:   msg.EventId,
			SenderId:  msg.SenderId,
			Timestamp: msg.Timestamp,
		}

		if msg.ContentBytes == nil {
			pbMess.Content = &pb.Message_Text{Text: msg.ContentText}
		} else {
			pbMess.Content = &pb.Message_Binary{Binary: msg.ContentBytes}
		}

		stream.Send(pbMess)
	}
	return nil
}

func (b *Broker) ListTopics(ctx context.Context, req *emptypb.Empty) (*pb.ListTopicsReply, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	topicInfos := make([]*pb.TopicInfo, 0, len(b.subscribers))
	for k, t := range b.subscribers {
		topicInfos = append(topicInfos, &pb.TopicInfo{Topic: k, SubscriberCount: int32(len(t))})
	}
	return &pb.ListTopicsReply{Topics: topicInfos}, nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	port := os.Getenv("PORT")
	tls := os.Getenv("TLS_ENABLED") == "true"
	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")

	if port == "" {
		log.Fatal("PORT environment variable is not set")
	}
	if tls && certFile == "" {
		log.Fatal("TLS_CERT_FILE environment variable is not set or set TLS_ENABLED to false without providing a certificate file")
	}
	if tls && keyFile == "" {
		log.Fatal("TLS_KEY_FILE environment variable is not set or set TLS_ENABLED to false without providing a certificate file")
	}

	// Initialize gRPC server options
	opts := []grpc.ServerOption{}

	if tls {
		creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)

		if err != nil {
			log.Fatalf("Failed loading certificates: %v\n", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	server := grpc.NewServer(opts...)
	broker := &Broker{subscribers: make(map[string][]chan SubscriberMessage)}
	pb.RegisterMessageBrokerServer(server, broker)

	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Println("Message broker running on port", port)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
