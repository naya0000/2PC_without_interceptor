package main

import (
	"context"
	"net"
	"os"
	"reflect"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v2"

	airlinepb "github.com/naya0000/2PC_without_interceptor/proto/airlineBooking"
	coordinatorpb "github.com/naya0000/2PC_without_interceptor/proto/coordinator"
	log "github.com/sirupsen/logrus"

	"google.golang.org/grpc"
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
	Address    string `yaml:"address"`
	Service    string `yaml:"service"`
	Method     string `yaml:"method"`
	PrimaryKey string `yaml:"primary_key"`
}

type CoordinatorServer struct {
	coordinatorpb.UnimplementedTransactionCoordinatorServer
	Config *Config
}

var (
	// 儲存 gRPC 連線池
	clientCache sync.Map
)

var proposeTimeout = 5 * time.Second // 預設值

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

// 取得 gRPC 連線（使用連線池）
func getGrpcConnection(address string) (*grpc.ClientConn, error) {
	if conn, ok := clientCache.Load(address); ok {
		return conn.(*grpc.ClientConn), nil
	}

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	clientCache.Store(address, conn)
	return conn, nil
}

func (s *CoordinatorServer) StartTransaction(ctx context.Context, req *coordinatorpb.StartTransactionRequest) (*coordinatorpb.StartTransactionResponse, error) {
	// log.Printf("Starting transactionId: %s with operations: %v", req.TxId, req.Operations)

	// 紀錄成功 Propose 的操作，方便取消
	var proposedOperations []ProposedOperation
	var availableSeats []int32

	for _, operation := range req.Operations {
		for i, svc := range s.Config.Services {
			if operation.Address == svc.Address && operation.Service == svc.Service && operation.Method == svc.Method {
				// 1. Propose Phase
				proposeCtx, cancel := context.WithTimeout(context.Background(), proposeTimeout)
				defer cancel()
				success, _ := s.executeOperation(proposeCtx, operation, "propose", req.TxId)
				if success {
					proposedOperation := ProposedOperation{Operation: operation}
					proposedOperations = append(proposedOperations, proposedOperation)
					// log.Printf("proposedOperations: %v", proposedOperations)
					break
				} else {
					log.Printf("[Propose Phase] Operation failed, entering Cancel Phase")

					// 2-(b). Cancel Phase：取消已 Propose 的操作
					for _, cancelOp := range proposedOperations {
						log.Printf("[Cancel Phase] Cancelling operation: Service=%s Method=%s", cancelOp.Operation.Service, cancelOp.Operation.Method)
						cancelCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
						defer cancel()
						success, _ := s.executeOperation(cancelCtx, cancelOp.Operation, "cancel", req.TxId)
						if success {
							log.Printf("Operation %s cancelled successfully", cancelOp.Operation.Method)
						} else {
							log.Errorf("Operation %s failed to cancel", cancelOp.Operation.Method)
						}
					}

					return &coordinatorpb.StartTransactionResponse{
						Success: false,
						Message: "Transaction failed in propose phase for operation: " + operation.Service + " " + operation.Method,
					}, nil
				}
			}
			// Operation that don't need to do 2PC
			if i == len(s.Config.Services)-1 {
				readCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				success, seatCount := s.executeOperation(readCtx, operation, "", req.TxId)
				if success {
					log.Printf("Operation succeeded: Service=%s, Method=%s", operation.Service, operation.Method)
					availableSeats = append(availableSeats, seatCount)
				} else {
					log.Printf("Operation failed, entering Cancel Phase")

					// 2-(b). Cancel Phase：取消已 Propose 的操作
					for _, cancelOp := range proposedOperations {
						cancelCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
						defer cancel()
						log.Printf("[Cancel Phase] Cancelling operation: Service=%s Method=%s", cancelOp.Operation.Service, cancelOp.Operation.Method)
						s.executeOperation(cancelCtx, cancelOp.Operation, "cancel", req.TxId)
					}

					return &coordinatorpb.StartTransactionResponse{
						Success: false,
						Message: "Transaction failed in operation: " + operation.Service + " " + operation.Method,
					}, nil
				}
			}
		}
	}

	for _, operation := range proposedOperations {
		// 2-(a). Commit Phase
		commitCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second) // 即使超時，也必須強制送出 commit，保證一致性
		defer cancel()
		success, seatCount := s.executeOperation(commitCtx, operation.Operation, "commit", req.TxId)
		if !success {
			log.Errorf("[Commit Phase] Operation failed.")
			return &coordinatorpb.StartTransactionResponse{
				Success: false,
				Message: "Transaction failed in Commit Phase",
			}, nil
		}
		availableSeats = append(availableSeats, seatCount)
	}

	log.Printf("Transaction %s completed successfully", req.TxId)
	return &coordinatorpb.StartTransactionResponse{
		Success:    true,
		Message:    "Transaction committed successfully",
		RespValues: availableSeats,
	}, nil
}

func (s *CoordinatorServer) executeOperation(ctx context.Context, operation *coordinatorpb.Operation, phase string, txId string) (bool, int32) {
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

	// 取得 gRPC 連線（從連線池）
	conn, err := getGrpcConnection(operation.Address)
	if err != nil {
		log.Printf("Failed to connect to service: %v", err)
		return false, -1
	}

	client := airlinepb.NewFlightBookingClient(conn)

	method := reflect.ValueOf(client).MethodByName(methodName)
	if !method.IsValid() {
		log.Printf("Method %s not found", methodName)
		return false, -1
	}

	reqType := method.Type().In(1).Elem()
	// log.Printf("Request type: %v", reqType)
	req := reflect.New(reqType).Interface()
	// log.Printf("Operation: %v", operation)

	if err := proto.Unmarshal(operation.Request, req.(proto.Message)); err != nil {
		log.Printf("Unmarshal error: %v", err)
		return false, -1
	}
	// log.Printf("Request: %v", req)
	reflect.ValueOf(req).Elem().FieldByName("TxId").SetString(txId)
	// log.Printf("Request: %v", req)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// 檢查 context 是否已取消
		if ctx.Err() != nil {
			log.Printf("[%s Phase] Operation %s aborted due to context error: %v (attempt %d)", phase, operation.Method, ctx.Err(), attempt)
			return false, -1
		}

		// 調用 gRPC 方法
		results := method.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(req),
		})

		if len(results) == 2 && !results[1].IsNil() {
			err := results[1].Interface().(error)
			log.Printf("[%s Phase] %s failed (attempt %d): %v", phase, operation.Method, attempt, err)
			time.Sleep(retryInterval)
			continue
		}
		respValue := reflect.ValueOf(results[0].Interface())
		// log.Printf("Response: %v", respValue)
		// log.Printf("respValue.Field(0):%v", respValue.Field(0))

		if respValue.Kind() == reflect.Ptr {
			if respValue.IsNil() {
				log.Printf("Method %s returned nil response", operation.Method)
				return false, -1
			}
			respValue = respValue.Elem()
		}

		if respValue.Kind() != reflect.Struct {
			log.Printf("Method %s did not return a struct, got: %v", operation.Method, respValue.Kind())
			return false, -1
		}
		// log.Printf("respValue: %v", respValue)

		availableSeatsField := respValue.FieldByName("AvailableSeats")
		// log.Printf("availableSeatsField: %v", availableSeatsField)
		// // log.Printf("AvailableSeats: %v", availableSeatsField)
		if availableSeatsField.IsValid() {
			// log.Printf("AvailableSeats: %v", availableSeatsField.Int())
			return true, int32(availableSeatsField.Int())
		}
	}
	log.Printf("[%s Phase] Operation %s failed after %d attempts", phase, operation.Method, maxRetries)
	return false, -1
}

func main() {
	// 動態設定 propose timeout
	timeoutStr := os.Getenv("PROPOSE_TIMEOUT")
	if timeoutStr != "" {
		if sec, err := strconv.Atoi(timeoutStr); err == nil {
			proposeTimeout = time.Duration(sec) * time.Second
			log.Infof("📡 Propose timeout set to %d seconds", sec)
		}
	}

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
