package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"os/exec"
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

	if err != nil { // timeout or other errors
		*results = append(*results, TransactionResult{TxID: txID, StartTime: start, EndTime: end, Success: false, Message: err.Error()})
		// log.Printf("Transaction %s failed: %v", txID, err)
	} else if resp.Success {
		*results = append(*results, TransactionResult{TxID: txID, StartTime: start, EndTime: end, Success: true, RespValues: resp.RespValues})
		// log.Printf("Transaction %s completed successfully", txID)
	} else {
		*results = append(*results, TransactionResult{TxID: txID, StartTime: start, EndTime: end, Success: false, Message: resp.Message})
		// log.Printf("Transaction %s failed: %s", txID, transactionResp.Message)
	}
}

func main() {
	initGrpcConnection()
	defer closeGrpcConnection()
	file, _ := os.Create("propose_timeout_experiment.csv")
	writer := csv.NewWriter(file)
	writer.Write([]string{"ProposeTimeout(s)", "SuccessRate(%)", "SuccessCount", "TPS", "AvgLatency(s)"})

	for _, timeout := range []int{2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30} {
		// for _, timeout := range []int{5} {
		log.Printf("Running with PROPOSE_TIMEOUT=%ds", timeout)

		cmd := exec.Command("go", "run", "../coordinator/coordinator.go")
		cmd.Env = append(os.Environ(), fmt.Sprintf("PROPOSE_TIMEOUT=%d", timeout))
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			log.Fatalf("Coordinator start failed: %v", err)
		}
		time.Sleep(3 * time.Second)

		flightId := "flight1"
		bookSeatsReq1 := &airlinepb.BookSeatsRequest{FlightId: flightId, SeatCount: 1}
		bookSeatsReqBytes1, _ := proto.Marshal(bookSeatsReq1)

		bookSeatsReq2 := &airlinepb.BookSeatsRequest{FlightId: flightId, SeatCount: 1}
		bookSeatsReqBytes2, _ := proto.Marshal(bookSeatsReq2)

		getSeatsReq1, _ := proto.Marshal(&airlinepb.GetSeatsRequest{FlightId: flightId})
		getSeatsReq2, _ := proto.Marshal(&airlinepb.GetSeatsRequest{FlightId: flightId})

		write1 := &coordinatorpb.Operation{Address: "192.168.0.125:50051", Service: "FlightBooking", Method: "BookSeats", Request: bookSeatsReqBytes1}
		write2 := &coordinatorpb.Operation{Address: "192.168.0.125:50052", Service: "FlightBooking", Method: "BookSeats", Request: bookSeatsReqBytes2}

		read1 := &coordinatorpb.Operation{Address: "192.168.0.125:50051", Service: "FlightBooking", Method: "GetSeats", Request: getSeatsReq1}
		read2 := &coordinatorpb.Operation{Address: "192.168.0.125:50052", Service: "FlightBooking", Method: "GetSeats", Request: getSeatsReq2}

		var wg sync.WaitGroup
		var mu sync.Mutex
		var results []TransactionResult

		transactions := make(map[string][]*coordinatorpb.Operation)
		transactionCount := 300

		for i := range transactionCount {
			if i%2 == 0 {
				txID := fmt.Sprintf("tx_write_%03d", i+1)
				transactions[txID] = []*coordinatorpb.Operation{write1, write2}
			} else {
				txID := fmt.Sprintf("tx_read_%03d", i+1)
				transactions[txID] = []*coordinatorpb.Operation{read1, read2}
			}
		}

		start := time.Now()
		for txID, ops := range transactions {
			time.Sleep(100 * time.Millisecond)
			wg.Add(1)
			go runTransaction(txID, ops, 30, &results, &mu, &wg)
		}
		wg.Wait()
		end := time.Now()

		successCount := 0
		var totalLatency time.Duration
		for _, res := range results {
			if res.Success {
				successCount++
				totalLatency += res.EndTime.Sub(res.StartTime)
			}
			// log.Printf("Transaction %s: %v, %v, %s", res.TxID, res.Success, res.RespValues, res.Message)
			if res.Success && res.RespValues[0] != res.RespValues[1] {
				log.Printf("❌ %s: %v, %v, %s", res.TxID, res.Success, res.RespValues, res.Message)
			} else {
				log.Printf("✅ %s: %v, %v, %s", res.TxID, res.Success, res.RespValues, res.Message)
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
			fmt.Sprintf("%d", timeout),
			fmt.Sprintf("%.2f", successRate),
			fmt.Sprintf("%d", successCount),
			fmt.Sprintf("%.2f", tps),
			fmt.Sprintf("%.3f", avgLatency),
		})
		writer.Flush()

		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
			log.Printf("❌ Failed to kill coordinator: %v", err)
		} else {
			log.Printf("🧹 Coordinator (PID %d) killed", cmd.Process.Pid)
		}

		time.Sleep(5 * time.Second)
	}
	log.Println("✅ 實驗完成，結果寫入 propose_timeout_experiment.csv")
}
