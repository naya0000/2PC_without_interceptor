package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	airlinepb "github.com/naya0000/2PC_without_interceptor/proto/airlineBooking"
	coordinatorpb "github.com/naya0000/2PC_without_interceptor/proto/coordinator"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

type TransactionResult struct {
	TxID       string
	StartTime  time.Time
	EndTime    time.Time
	Success    bool
	RespValues []int32
	message    string
}

var (
	coordinatorClient coordinatorpb.TransactionCoordinatorClient
	grpcConn          *grpc.ClientConn
)

func initGrpcConnection() {
	var err error
	grpcConn, err = grpc.NewClient("localhost:50050", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Coordinator: %v", err)
	}
	coordinatorClient = coordinatorpb.NewTransactionCoordinatorClient(grpcConn)
}

func closeGrpcConnection() {
	grpcConn.Close()
}

func startTransaction(txID string, operations []*coordinatorpb.Operation, sec int, wg *sync.WaitGroup, results *[]TransactionResult, mu *sync.Mutex) {
	defer wg.Done()

	transactionReq := &coordinatorpb.StartTransactionRequest{
		TxId:       txID,
		Operations: operations,
	}
	timeout := time.Duration(sec) * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	startTime := time.Now()
	transactionResp, err := coordinatorClient.StartTransaction(ctx, transactionReq)
	endTime := time.Now()

	mu.Lock()
	defer mu.Unlock()
	if err != nil { // timeout or other errors
		*results = append(*results, TransactionResult{TxID: txID, StartTime: startTime, EndTime: endTime, Success: false, message: err.Error()})
		// log.Printf("Transaction %s failed: %v", txID, err)
	} else if transactionResp.Success {
		*results = append(*results, TransactionResult{TxID: txID, StartTime: startTime, EndTime: endTime, Success: true, RespValues: transactionResp.RespValues})
		// log.Printf("Transaction %s completed successfully", txID)
	} else {
		*results = append(*results, TransactionResult{TxID: txID, StartTime: startTime, EndTime: endTime, Success: false, message: transactionResp.Message})
		// log.Printf("Transaction %s failed: %s", txID, transactionResp.Message)
	}
}

func runExperiment() {

	flightId1 := "flight1"

	bookSeatsReq1 := &airlinepb.BookSeatsRequest{FlightId: flightId1, SeatCount: 1}
	bookSeatsReqBytes1, _ := proto.Marshal(bookSeatsReq1)

	bookSeatsReq2 := &airlinepb.BookSeatsRequest{FlightId: flightId1, SeatCount: 1}
	bookSeatsReqBytes2, _ := proto.Marshal(bookSeatsReq2)

	getSeatsReq1 := &airlinepb.GetSeatsRequest{FlightId: flightId1}
	getSeatsReqBytes1, _ := proto.Marshal(getSeatsReq1)

	getSeatsReq2 := &airlinepb.GetSeatsRequest{FlightId: flightId1}
	getSeatsReqBytes2, _ := proto.Marshal(getSeatsReq2)

	// write1 := &coordinatorpb.Operation{Address: "192.168.0.126:50051", Service: "FlightBooking", Method: "BookSeats", Request: bookSeatsReqBytes1}
	// write2 := &coordinatorpb.Operation{Address: "192.168.0.126:50052", Service: "FlightBooking", Method: "BookSeats", Request: bookSeatsReqBytes2}

	write1 := &coordinatorpb.Operation{Address: "localhost:50051", Service: "FlightBooking", Method: "BookSeats", Request: bookSeatsReqBytes1}
	write2 := &coordinatorpb.Operation{Address: "localhost:50052", Service: "FlightBooking", Method: "BookSeats", Request: bookSeatsReqBytes2}

	read1 := &coordinatorpb.Operation{Address: "localhost:50051", Service: "FlightBooking", Method: "GetSeats", Request: getSeatsReqBytes1}
	read2 := &coordinatorpb.Operation{Address: "localhost:50052", Service: "FlightBooking", Method: "GetSeats", Request: getSeatsReqBytes2}

	// transactionCount := 200
	transactionCount := 100

	// 建立 CSV 檔案
	f, err := os.Create("timeout_experiment.csv")
	if err != nil {
		log.Fatal("create file:", err)
	}
	defer f.Close()
	writer := csv.NewWriter(f)
	writer.Write([]string{"Timeout(s)", "SuccessRate(%)", "SuccessCount", "TPS", "AvgLatency(s)"})

	// for sec := 1; sec < 20; sec += 2 {
	sec := 30
	log.Printf("Running with timeout: %ds", sec)
	var results []TransactionResult
	var wg sync.WaitGroup
	var mu sync.Mutex

	transactions := make(map[string][]*coordinatorpb.Operation)

	for i := range transactionCount {
		if i%2 == 0 {
			txID := fmt.Sprintf("tx_write_%03d", i+1)
			// if i%4 == 0 {
			// 	transactions[txID] = []*coordinatorpb.Operation{write2, write1}
			// } else {
			transactions[txID] = []*coordinatorpb.Operation{write1, write2}
			// }
		} else {
			txID := fmt.Sprintf("tx_read_%03d", i+1)
			transactions[txID] = []*coordinatorpb.Operation{read1, read2}
		}
	}

	start := time.Now()
	for txID, ops := range transactions {
		time.Sleep(100 * time.Millisecond)
		wg.Add(1)
		go startTransaction(txID, ops, sec, &wg, &results, &mu)
	}
	wg.Wait()
	end := time.Now()

	// 統計
	successCount := 0
	var totalLatency time.Duration

	for _, res := range results {
		if res.Success {
			successCount++
			totalLatency += res.EndTime.Sub(res.StartTime)
		}
		if res.Success && res.RespValues[0] != res.RespValues[1] {
			log.Printf("❌ %s: %v, %v, %s", res.TxID, res.Success, res.RespValues, res.message)
		} else {
			log.Printf("✅ %s: %v, %v, %s", res.TxID, res.Success, res.RespValues, res.message)
		}
	}

	successRate := float64(successCount) / float64(transactionCount) * 100
	tps := float64(successCount) / end.Sub(start).Seconds()
	avgLatency := 0.0
	if successCount > 0 {
		avgLatency = totalLatency.Seconds() / float64(successCount)
	}

	log.Printf("SuccessRate=%.2f%%, SuccessCount=%d, TPS=%.2f, AvgLatency=%.3fs",
		successRate, successCount, tps, avgLatency)

	writer.Write([]string{
		fmt.Sprintf("%d", sec),
		fmt.Sprintf("%.2f", successRate),
		fmt.Sprintf("%d", successCount),
		fmt.Sprintf("%.2f", tps),
		fmt.Sprintf("%.3f", avgLatency),
	})
	writer.Flush()
	// }
	log.Println("✅ 實驗完成，結果已寫入 timeout_experiment.csv")
}

func main() {
	initGrpcConnection()
	defer closeGrpcConnection()

	// 測試 n 筆交易
	// for _, txSize := range []int{1} {
	// 	runTest(txSize)
	// }

	runExperiment()

	log.Println("All transactions completed.")
}
