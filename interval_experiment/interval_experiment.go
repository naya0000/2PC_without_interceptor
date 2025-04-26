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

func runTransaction(txID string, operations []*coordinatorpb.Operation, sec int, results *[]TransactionResult, mu *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sec)*time.Second)
	defer cancel()

	req := &coordinatorpb.StartTransactionRequest{
		TxId:       txID,
		Operations: operations,
	}

	start := time.Now()
	resp, err := coordinatorClient.StartTransaction(ctx, req)
	end := time.Now()

	mu.Lock()
	defer mu.Unlock()

	if err != nil {
		*results = append(*results, TransactionResult{TxID: txID, StartTime: start, EndTime: end, Success: false, Message: err.Error()})
	} else if resp.Success {
		*results = append(*results, TransactionResult{TxID: txID, StartTime: start, EndTime: end, Success: true, RespValues: resp.RespValues})
	} else {
		*results = append(*results, TransactionResult{TxID: txID, StartTime: start, EndTime: end, Success: false, Message: resp.Message})
	}
}

func main() {
	initGrpcConnection()
	defer closeGrpcConnection()

	file, err := os.Create("interval_experiment.csv")
	if err != nil {
		log.Fatalf("Failed to create CSV file: %v", err)
	}
	writer := csv.NewWriter(file)
	writer.Write([]string{"Interval(ms)", "SuccessRate(%)", "SuccessCount", "TPS", "AvgLatency(s)", "Latency Outlier Count"})

	// 測試發送間隔 ms
	for _, interval := range []int{50, 100, 150, 200, 250, 300, 350} {
		log.Printf("Running with send interval = %d ms", interval)

		// 啟動 Coordinator
		cmd := exec.Command("go", "run", "../coordinator/coordinator.go")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			log.Fatalf("Coordinator start failed: %v", err)
		}
		time.Sleep(3 * time.Second)

		// 準備兩個 BookSeats 操作
		flightID := "flight1"
		req1 := &airlinepb.BookSeatsRequest{FlightId: flightID, SeatCount: 1}
		req2 := &airlinepb.BookSeatsRequest{FlightId: flightID, SeatCount: 1}
		bytes1, _ := proto.Marshal(req1)
		bytes2, _ := proto.Marshal(req2)

		op1 := &coordinatorpb.Operation{Address: "192.168.0.125:50051", Service: "FlightBooking", Method: "BookSeats", Request: bytes1}
		op2 := &coordinatorpb.Operation{Address: "192.168.0.125:50052", Service: "FlightBooking", Method: "BookSeats", Request: bytes2}

		// 建立交易清單
		transactionCount := 500
		transactions := make(map[string][]*coordinatorpb.Operation, transactionCount)
		for i := 0; i < transactionCount; i++ {
			txID := fmt.Sprintf("tx_%04d", i+1)

			// if i%2 == 0 {
			transactions[txID] = []*coordinatorpb.Operation{op1, op2}
			// } else {
			// 	transactions[txID] = []*coordinatorpb.Operation{op2, op1}
			// }
		}

		// 並行發送交易
		var wg sync.WaitGroup
		var mu sync.Mutex
		var results []TransactionResult

		// startAll := time.Now()
		for txID, ops := range transactions {
			time.Sleep(time.Duration(interval) * time.Millisecond) // 可變間隔
			wg.Add(1)
			go runTransaction(txID, ops, 5, &results, &mu, &wg)
		}
		wg.Wait()
		// endAll := time.Now()

		// 統計
		successCount := 0
		var latencies []float64
		for _, res := range results {
			if res.Success {
				successCount++
				latency := res.EndTime.Sub(res.StartTime).Seconds()
				latencies = append(latencies, latency)
			}
			if res.Success && res.RespValues[0] != res.RespValues[1] {
				log.Printf("❌ %s: %v, %v, %s", res.TxID, res.Success, res.RespValues, res.Message)
			} else {
				log.Printf("✅ %s: %v, %v, %s", res.TxID, res.Success, res.RespValues, res.Message)
			}
		}

		// 隔離成功交易時間窗，計算純 TPS
		var firstStart, lastEnd time.Time
		for _, r := range results {
			if !r.Success {
				continue
			}
			if firstStart.IsZero() || r.StartTime.Before(firstStart) {
				firstStart = r.StartTime
			}
			if lastEnd.IsZero() || r.EndTime.After(lastEnd) {
				lastEnd = r.EndTime
			}
		}
		pureElapsed := lastEnd.Sub(firstStart)

		successRate := float64(successCount) / float64(transactionCount) * 100
		// totalSendDelay := time.Duration(interval*(transactionCount-1)) * time.Millisecond
		tps := float64(successCount) / pureElapsed.Seconds()
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
		log.Printf("Interval=%dms → SuccessRate=%.2f%%, SuccessCount=%d, TPS=%.2f, AvgLatency=%.3fs, OutlierCount=%d",
			interval, successRate, successCount, tps, avgLatency, outlierCount)

		// 寫入 CSV
		writer.Write([]string{
			fmt.Sprintf("%d", interval),
			fmt.Sprintf("%.2f", successRate),
			fmt.Sprintf("%d", successCount),
			fmt.Sprintf("%.2f", tps),
			fmt.Sprintf("%.3f", avgLatency),
			fmt.Sprintf("%d", outlierCount),
		})
		writer.Flush()

		// 停掉 Coordinator
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
			log.Printf("❌ Failed to kill coordinator: %v", err)
		}
		time.Sleep(2 * time.Second)
	}

	log.Println("✅ 實驗完成，結果寫入 interval_experiment.csv")
}
