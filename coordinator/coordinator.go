package main

import (
	"context"
	"log"
	"net"
	"os"
	"reflect"
	"time"

	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v2"

	airlinepb "github.com/naya0000/2PC_without_interceptor/proto/airlineBooking"
	coordinatorpb "github.com/naya0000/2PC_without_interceptor/proto/coordinator"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const (
	maxRetries    = 10
	retryInterval = 1 * time.Second
)

type ProposedOperation struct {
	Operation *coordinatorpb.Operation
}

type Config struct {
	Services []ServiceConfig `yaml:"services"`
}

type ServiceConfig struct {
	Service    string `yaml:"service"`
	Method     string `yaml:"method"`
	PrimaryKey string `yaml:"primary_key"`
}

type CoordinatorServer struct {
	coordinatorpb.UnimplementedTransactionCoordinatorServer
	Config *Config
}

// 檢查錯誤是否可重試
func isRetryableError(err error) bool {
	// st, ok := status.FromError(err)
	// if !ok {
	// 	return false // 不是 gRPC 錯誤，無法判斷是否可重試
	// }

	// switch st.Code() {
	// case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
	// 	return true
	// default:
	// 	return false
	// }
	return true
}

func loadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (s *CoordinatorServer) StartTransaction(ctx context.Context, req *coordinatorpb.StartTransactionRequest) (*coordinatorpb.StartTransactionResponse, error) {
	log.Printf("Starting transactionId: %s", req.TxId)

	// 創建獨立的 context，避免共享問題
	newCtx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	newCtx = metadata.AppendToOutgoingContext(newCtx, "TxId", req.TxId)

	// 紀錄成功 Propose 的操作，方便取消
	var proposedOperations []ProposedOperation

	for _, operation := range req.Operations {
		for i, svc := range s.Config.Services {
			if operation.Service == svc.Service && operation.Method == svc.Method {
				// 1. Propose Phase
				success := s.executeOperation(newCtx, operation, "propose", req.TxId)
				if success {
					proposedOperation := ProposedOperation{Operation: operation}
					proposedOperations = append(proposedOperations, proposedOperation)
					log.Printf("proposedOperations: %v", proposedOperations)
					break
				} else {
					log.Printf("[Propose Phase] Operation failed, entering Cancel Phase")

					// 2-(b). Cancel Phase：取消已 Propose 的操作
					for _, cancelOp := range proposedOperations {
						log.Printf("[Cancel Phase] Cancelling operation: Service=%s Method=%s", cancelOp.Operation.Service, cancelOp.Operation.Method)
						s.executeOperation(newCtx, cancelOp.Operation, "cancel", req.TxId)
					}

					return &coordinatorpb.StartTransactionResponse{
						Success: false,
						Message: "Transaction failed in propose phase for operation: " + operation.Service + " " + operation.Method,
					}, nil
				}
			}
			// Operation that don't need to do 2PC
			if i == len(s.Config.Services)-1 {
				success := s.executeOperation(newCtx, operation, "", req.TxId)
				if success {
					log.Printf("Operation succeeded: Service=%s, Method=%s", operation.Service, operation.Method)
				} else {
					log.Printf("Operation failed, entering Cancel Phase")

					// 2-(b). Cancel Phase：取消已 Propose 的操作
					for _, cancelOp := range proposedOperations {
						log.Printf("[Cancel Phase] Cancelling operation: Service=%s Method=%s", cancelOp.Operation.Service, cancelOp.Operation.Method)
						s.executeOperation(newCtx, cancelOp.Operation, "cancel", req.TxId)
					}

					return &coordinatorpb.StartTransactionResponse{
						Success: false,
						Message: "Transaction failed in operation: " + operation.Service + " " + operation.Method,
					}, nil
				}
			}
		}
	}

	// 2-(a). Commit Phase
	for _, operation := range req.Operations {
		for _, svc := range s.Config.Services {
			if operation.Service == svc.Service && operation.Method == svc.Method {
				success := s.executeOperation(newCtx, operation, "commit", req.TxId)
				if !success {
					log.Printf("[Commit Phase] Operation failed.")
					return &coordinatorpb.StartTransactionResponse{
						Success: false,
						Message: "Transaction failed in Commit Phase",
					}, nil
				}
			}
		}
	}

	log.Printf("Transaction %s completed successfully", req.TxId)
	return &coordinatorpb.StartTransactionResponse{
		Success: true,
		Message: "Transaction committed successfully",
	}, nil
}

func (s *CoordinatorServer) executeOperation(ctx context.Context, operation *coordinatorpb.Operation, phase string, txId string) bool {
	methodName := ""
	if phase == "propose" {
		methodName = "Propose" + operation.Method
	} else if phase == "commit" {
		methodName = "Commit" + operation.Method
	} else if phase == "cancel" {
		methodName = "Cancel" + operation.Method
	} else {
		methodName = operation.Method
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		conn, err := grpc.NewClient(operation.Service, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("Failed to connect to service: %v", err)
			return false
		}
		defer conn.Close()
		client := airlinepb.NewFlightBookingClient(conn)

		method := reflect.ValueOf(client).MethodByName(methodName)
		if !method.IsValid() {
			log.Printf("Method %s not found", methodName)
			return false
		}

		reqType := method.Type().In(1).Elem()
		req := reflect.New(reqType).Interface()
		if err := proto.Unmarshal(operation.Request, req.(proto.Message)); err != nil {
			log.Printf("Unmarshal error: %v", err)
			return false
		}
		reflect.ValueOf(req).Elem().FieldByName("TxId").SetString(txId)

		results := method.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(req)})
		if len(results) == 2 && !results[1].IsNil() {
			log.Printf("[%s Phase] Operation failed: %v", phase, results[1].Interface())
			if attempt == maxRetries {
				return false
			}
			time.Sleep(retryInterval)
			continue
		}
		log.Printf("[%s Phase] Operation succeeded: Service %s Method=%s", phase, operation.Service, operation.Method)
		return true
	}
	return false
}

func main() {
	config, err := loadConfig("/Users/naya/mygo/src/2PC_without_interceptor/config/2PC_config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	listener, err := net.Listen("tcp", ":50050")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	coordinatorpb.RegisterTransactionCoordinatorServer(grpcServer, &CoordinatorServer{Config: config})

	log.Printf("Coordinator server listening on %v", listener.Addr())
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
