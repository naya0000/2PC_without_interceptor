package main

import (
	"context"
	"log"
	"sync"
	"time"

	airlinepb "github.com/naya0000/2PC_without_interceptor/proto/airlineBooking"
	coordinatorpb "github.com/naya0000/2PC_without_interceptor/proto/coordinator"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

func startTransaction(txID string, operations []*coordinatorpb.Operation, coordinatorClient coordinatorpb.TransactionCoordinatorClient, wg *sync.WaitGroup, mu *sync.Mutex, results *[]string) {
	defer wg.Done()

	transactionReq := &coordinatorpb.StartTransactionRequest{
		TxId:       txID,
		Operations: operations,
	}

	// 建立上下文和超時
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	// 發起交易
	transactionResp, err := coordinatorClient.StartTransaction(ctx, transactionReq)
	mu.Lock()
	defer mu.Unlock()
	if err != nil {
		log.Printf("Transaction %s failed: %v", txID, err)
		*results = append(*results, txID+": failed")
	} else {
		// log.Printf("Transaction %s response: %v", txID, transactionResp)
		*results = append(*results, txID+": "+transactionResp.Message)
	}
}

func main() {
	// 建立 gRPC 連接到 TransactionCoordinator
	conn, err := grpc.NewClient("localhost:50050", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Coordinator: %v", err)
	}
	defer conn.Close()
	coordinatorClient := coordinatorpb.NewTransactionCoordinatorClient(conn)

	flightId1 := "flight1"
	flightId2 := "flight2"
	flightId3 := "flight3"

	getSeatsReq1 := &airlinepb.GetSeatsRequest{FlightId: flightId1}
	getSeatsReqBytes1, _ := proto.Marshal(getSeatsReq1)

	getSeatsReq2 := &airlinepb.GetSeatsRequest{FlightId: flightId2}
	getSeatsReqBytes2, _ := proto.Marshal(getSeatsReq2)

	bookSeatsReq1 := &airlinepb.BookSeatsRequest{FlightId: flightId1, SeatCount: 2}
	bookSeatsReqBytes1, _ := proto.Marshal(bookSeatsReq1)

	bookSeatsReq2 := &airlinepb.BookSeatsRequest{FlightId: flightId2, SeatCount: 2}
	bookSeatsReqBytes2, _ := proto.Marshal(bookSeatsReq2)

	bookSeatsReq3 := &airlinepb.BookSeatsRequest{FlightId: flightId3, SeatCount: 2}
	bookSeatsReqBytes3, _ := proto.Marshal(bookSeatsReq3)

	op1 := &coordinatorpb.Operation{Service: "localhost:50051", Method: "GetSeats", Request: getSeatsReqBytes1}
	op2 := &coordinatorpb.Operation{Service: "localhost:50052", Method: "GetSeats", Request: getSeatsReqBytes2}
	op3 := &coordinatorpb.Operation{Service: "localhost:50051", Method: "BookSeats", Request: bookSeatsReqBytes1}
	op4 := &coordinatorpb.Operation{Service: "localhost:50052", Method: "BookSeats", Request: bookSeatsReqBytes2}
	op5 := &coordinatorpb.Operation{Service: "localhost:50052", Method: "BookSeats", Request: bookSeatsReqBytes3}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []string

	transactions := map[string][]*coordinatorpb.Operation{
		"tx001": {op3, op4, op1},
		"tx002": {op3, op4, op5, op2},
		// "tx003": {op1, op2, op3, op4},
	}

	for txID, ops := range transactions {
		wg.Add(1)
		go startTransaction(txID, ops, coordinatorClient, &wg, &mu, &results)
	}

	wg.Wait()

	log.Println("All transactions completed. Results:")
	for _, result := range results {
		log.Println(result)
	}
}
