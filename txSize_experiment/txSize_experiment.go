package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"sync"
	"syscall"
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
	Message    string
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
		*results = append(*results, TransactionResult{TxID: txID, StartTime: startTime, EndTime: endTime, Success: false, Message: err.Error()})
		// log.Printf("Transaction %s failed: %v", txID, err)
	} else if transactionResp.Success {
		*results = append(*results, TransactionResult{TxID: txID, StartTime: startTime, EndTime: endTime, Success: true, RespValues: transactionResp.RespValues})
		// log.Printf("Transaction %s completed successfully", txID)
	} else {
		*results = append(*results, TransactionResult{TxID: txID, StartTime: startTime, EndTime: endTime, Success: false, Message: transactionResp.Message})
		// log.Printf("Transaction %s failed: %s", txID, transactionResp.Message)
	}
}

func main() {
	initGrpcConnection()
	defer closeGrpcConnection()
	file, _ := os.Create("txSize_experiment.csv")
	writer := csv.NewWriter(file)
	writer.Write([]string{"TransactionSize", "SuccessRate(%)", "SuccessCount", "TPS", "AvgLatency(s)", "Latency Outlier Count"})

	// 測試 n 筆交易
	// for _, txSize := range []int{10, 20, 30, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500} {
	for _, txSize := range []int{10, 20, 30, 50, 100, 150} {
		log.Printf("Running with transaction size: %d", txSize)

		cmd := exec.Command("go", "run", "../coordinator/coordinator.go")
		cmd.Env = append(os.Environ(), fmt.Sprintf("txSize=%d", txSize))
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			log.Fatalf("Coordinator start failed: %v", err)
		}
		time.Sleep(3 * time.Second)

		flightId1 := "flight1"

		getSeatsReq1 := &airlinepb.GetSeatsRequest{FlightId: flightId1}
		getSeatsReqBytes1, _ := proto.Marshal(getSeatsReq1)

		getSeatsReq2 := &airlinepb.GetSeatsRequest{FlightId: flightId1}
		getSeatsReqBytes2, _ := proto.Marshal(getSeatsReq2)

		bookSeatsReq1 := &airlinepb.BookSeatsRequest{FlightId: flightId1, SeatCount: 1}
		bookSeatsReqBytes1, _ := proto.Marshal(bookSeatsReq1)

		bookSeatsReq2 := &airlinepb.BookSeatsRequest{FlightId: flightId1, SeatCount: 1}
		bookSeatsReqBytes2, _ := proto.Marshal(bookSeatsReq2)
		// 192.168.0.126
		op1 := &coordinatorpb.Operation{Address: "192.168.0.125:50051", Service: "FlightBooking", Method: "GetSeats", Request: getSeatsReqBytes1}
		op2 := &coordinatorpb.Operation{Address: "192.168.0.125:50052", Service: "FlightBooking", Method: "GetSeats", Request: getSeatsReqBytes2}
		op3 := &coordinatorpb.Operation{Address: "192.168.0.125:50051", Service: "FlightBooking", Method: "BookSeats", Request: bookSeatsReqBytes1}
		op4 := &coordinatorpb.Operation{Address: "192.168.0.125:50052", Service: "FlightBooking", Method: "BookSeats", Request: bookSeatsReqBytes2}

		transactions := make(map[string][]*coordinatorpb.Operation)
		var wg sync.WaitGroup
		var results []TransactionResult
		var mu sync.Mutex

		for i := range txSize {
			if i%2 == 0 {
				txID := fmt.Sprintf("tx_write_%03d", i+1)
				transactions[txID] = []*coordinatorpb.Operation{op3, op4}
			} else {
				txID := fmt.Sprintf("tx_read_%03d", i+1)
				transactions[txID] = []*coordinatorpb.Operation{op1, op2}
			}
		}

		start := time.Now()
		for txID, ops := range transactions {
			time.Sleep(100 * time.Millisecond) // 模擬延遲
			wg.Add(1)
			go startTransaction(txID, ops, 30, &wg, &results, &mu)
		}
		wg.Wait()
		end := time.Now()

		// 計算 latency 和 TPS
		var latencies []float64
		successCount := 0
		for _, res := range results {
			if res.Success {
				successCount++
				latency := res.EndTime.Sub(res.StartTime).Seconds()
				latencies = append(latencies, latency)
			}
			// log.Printf("%s: %v, %v, %s, %s, %v, %s", res.TxID, res.Success, res.EndTime.Sub(res.StartTime), res.StartTime.Format("15:04:05.000"), res.EndTime.Format("15:04:05.000"), res.RespValues, res.message)
			if res.Success && res.RespValues[0] != res.RespValues[1] {
				log.Printf("❌ %s: %v, %v, %s", res.TxID, res.Success, res.RespValues, res.Message)
			} else {
				log.Printf("✅ %s: %v, %v, %s", res.TxID, res.Success, res.RespValues, res.Message)
			}
		}
		successRate := float64(successCount) / float64(txSize) * 100
		tps := float64(successCount) / end.Sub(start).Seconds()
		// 排除偏差值後再計算平均
		avgLatency := 0.0
		outlierCount := 0
		if len(latencies) > 0 {
			sort.Float64s(latencies)
			q1 := latencies[len(latencies)/4]
			q3 := latencies[(len(latencies)*3)/4]
			iqr := q3 - q1
			lowerBound := q1 - 1.5*iqr
			upperBound := q3 + 1.5*iqr

			var filtered []float64
			for _, v := range latencies {
				if v >= lowerBound && v <= upperBound {
					filtered = append(filtered, v)
				}
			}

			var sum float64
			for _, v := range filtered {
				sum += v
			}
			if len(filtered) > 0 {
				avgLatency = sum / float64(len(filtered))
			}
			outlierCount = len(latencies) - len(filtered)
			log.Printf("Removed %d outliers from latency", outlierCount)
		}

		// avgLatency := 0.0
		// if successCount > 0 {
		// 	avgLatency = totalLatency.Seconds() / float64(successCount)
		// }
		log.Printf("SuccessRate=%.2f%%, SuccessCount=%d, TPS=%.2f, AvgLatency=%.3fs, OutlierCount=%d",
			successRate, successCount, tps, avgLatency, outlierCount)

		writer.Write([]string{
			fmt.Sprintf("%d", txSize),
			fmt.Sprintf("%.2f", successRate),
			fmt.Sprintf("%d", successCount),
			fmt.Sprintf("%.2f", tps),
			fmt.Sprintf("%.3f", avgLatency),
			fmt.Sprintf("%d", outlierCount),
		})
		writer.Flush()

		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
			log.Printf("❌ Failed to kill coordinator: %v", err)
		} else {
			log.Printf("🧹 Coordinator (PID %d) killed", cmd.Process.Pid)
		}

		time.Sleep(5 * time.Second)
	}

	log.Println("✅ 實驗完成，結果寫入 txSize_experiment.csv")
}
