package main

import (
	"context"
	"flag"
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
	subscribers map[string]map[string]chan SubscriberMessage
	mu          sync.RWMutex
}

func (b *Broker) Publish(ctx context.Context, msg *pb.Message) (*pb.Response, error) {
	if msg.GetSenderId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "senderId is required")
	}

	subscribers := b.subscribers[msg.Topic]
	if len(subscribers) == 0 {
		return nil, status.Error(codes.NotFound, "no subscribers in this topic")
	}

	log.Printf("[Publish    ] topic: %s, sender: %s, eventId: %v, Subscribers: %v",
		msg.Topic, msg.GetSenderId(), msg.GetEventId(), len(subscribers))

	b.mu.RLock()
	for _, ch := range subscribers {
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

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

func (b *Broker) Subscribe(req *pb.SubscriptionRequest, stream pb.MessageBroker_SubscribeServer) error {
	// Return an error if the topic is empty or topic length is greater than 100
	if req.Topic == "" || len(req.Topic) > 100 {
		return status.Errorf(codes.InvalidArgument, "invalid topic: %s", req.Topic)
	}

	subId := req.GetSubscriberId()

	if subId == "" {
		return status.Errorf(codes.InvalidArgument, "subscriberId is required")
	}

	if b.subscribers[req.Topic][subId] != nil {
		return status.Errorf(codes.AlreadyExists, "subscriberId already exists")
	}

	log.Printf("[Subscribe  ] topic: %s, subscriber: %s", req.Topic, subId)

	ch := make(chan SubscriberMessage)
	b.mu.Lock()
	if len(b.subscribers[req.Topic]) == 0 {
		b.subscribers[req.Topic] = make(map[string]chan SubscriberMessage)
	}
	b.subscribers[req.Topic][subId] = ch
	b.mu.Unlock()

	ctx := stream.Context()
	ctx = context.WithValue(ctx, contextKey("subscriberId"), subId)
	go func() {
		<-ctx.Done()
		subId := ctx.Value(contextKey("subscriberId")).(string)
		log.Printf("[UnSubscribe] topic: %s, subscriber: %s", req.Topic, subId)

		b.mu.Lock()
		// Delete subscriber when disconnected
		delete(b.subscribers[req.Topic], subId)
		b.mu.Unlock()
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

func GetTopicSubscribers(topic map[string]chan SubscriberMessage) []string {
	subscriberIds := []string{}
	for id, _ := range topic {
		subscriberIds = append(subscriberIds, id)
	}

	return subscriberIds
}

func (b *Broker) ListTopics(ctx context.Context, req *pb.ListTopicsRequest) (*pb.ListTopicsReply, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	reqTopic := req.GetTopic()
	if reqTopic != "" {
		subscribers, ok := b.subscribers[reqTopic]
		if !ok {
			return nil, status.Error(codes.NotFound, "topic is empty")
		}

		topic := &pb.TopicInfo{
			Topic:         reqTopic,
			SubscriberIds: GetTopicSubscribers(subscribers),
		}

		return &pb.ListTopicsReply{
			Topics: append([]*pb.TopicInfo{}, topic),
		}, nil
	}

	topicInfos := make([]*pb.TopicInfo, 0, len(b.subscribers))
	for k, t := range b.subscribers {
		if len(t) == 0 {
			continue
		}

		topicInfos = append(topicInfos, &pb.TopicInfo{
			Topic:         k,
			SubscriberIds: GetTopicSubscribers(t),
		})
	}
	return &pb.ListTopicsReply{Topics: topicInfos}, nil
}

func main() {
	port, logFile, tls, certFile, keyFile := LoadEnvVariables()
	// Set up logging
	SetLogFile(logFile)

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
	broker := &Broker{subscribers: make(map[string]map[string]chan SubscriberMessage)}
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

func LoadEnvVariables() (string, string, bool, string, string) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	portFlag := flag.String("port", os.Getenv("PORT"), "Port number")
	logFileFlag := flag.String("logFile", os.Getenv("LOG_FILE"), "Log file path")
	tlsEnabledFlag := flag.String("tlsEnabled", os.Getenv("TLS_ENABLED"), "Enable TLS (true/false)")
	tlsCertFileFlag := flag.String("tlsCertFile", os.Getenv("TLS_CERT_FILE"), "TLS certificate file path")
	tlsKeyFileFlag := flag.String("tlsKeyFile", os.Getenv("TLS_KEY_FILE"), "TLS key file path")

	flag.Parse()

	port := *portFlag
	logFile := *logFileFlag
	tls := *tlsEnabledFlag == "true" // Check if TLS is enabled and get the certificate files
	certFile := *tlsCertFileFlag
	keyFile := *tlsKeyFileFlag

	if port == "" {
		port = ":50051" // Default port if not set
	}
	if tls && certFile == "" {
		log.Fatal("TLS_CERT_FILE environment variable is not set or set TLS_ENABLED to false without providing a certificate file")
	}
	if tls && keyFile == "" {
		log.Fatal("TLS_KEY_FILE environment variable is not set or set TLS_ENABLED to false without providing a certificate file")
	}

	return port, logFile, tls, certFile, keyFile
}

func SetLogFile(logFile string) {
	if logFile != "" {
		logFile, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		log.SetOutput(logFile)
	}
	log.SetFlags(log.Ldate | log.Lmicroseconds)
}
