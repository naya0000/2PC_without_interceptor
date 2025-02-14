package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"

	_ "github.com/lib/pq"
	airlinepb "github.com/naya0000/2PC_without_interceptor/proto/airlineBooking"
	"google.golang.org/grpc"
)

type FlightBookingServer struct {
	airlinepb.UnimplementedFlightBookingServer
	db *sql.DB
}

type Transaction struct {
	TxID       string
	PrimaryKey string
	Status     string
}

// 初始化 PostgreSQL 連接
func NewFlightBookingServer(db *sql.DB) *FlightBookingServer {
	return &FlightBookingServer{db: db}
}

func createTxTable(db *sql.DB) error {
	_, _ = db.Exec("DROP TABLE IF EXISTS transactions")
	createTableQuery := `
	CREATE TABLE transactions (
		tx_id VARCHAR(255),
		primary_key VARCHAR(255),
		status VARCHAR(50),
		PRIMARY KEY (tx_id, primary_key)
	);`
	_, err := db.Exec(createTableQuery)
	return err
}

func createFlightsTable(db *sql.DB) error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS flights (
	    flight_id VARCHAR(255) PRIMARY KEY,
	    available_seat INT NOT NULL CHECK (available_seat >= 0)
	);`
	_, err := db.Exec(createTableQuery)
	if err != nil {
		return err
	}

	// 插入測試數據（如果表是空的）
	insertDataQuery := `
	INSERT INTO flights (flight_id, available_seat) 
	SELECT 'flight1', 100 WHERE NOT EXISTS (SELECT 1 FROM flights WHERE flight_id = 'flight1');
	INSERT INTO flights (flight_id, available_seat) 
	SELECT 'flight2', 200 WHERE NOT EXISTS (SELECT 1 FROM flights WHERE flight_id = 'flight2');
	INSERT INTO flights (flight_id, available_seat) 
	SELECT 'flight3', 300 WHERE NOT EXISTS (SELECT 1 FROM flights WHERE flight_id = 'flight3');
	`
	_, err = db.Exec(insertDataQuery)
	return err
}

// 透過 PrimaryKey 檢查資源是否被其他 tx 鎖定
func (s *FlightBookingServer) IsLocked(primaryKey string) bool {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM transactions WHERE primary_key = $1 AND status = 'prepared'", primaryKey).Scan(&count)
	if err != nil {
		log.Printf("Failed to check if resource is locked: %v", err)
		return false
	}
	return count > 0
}

// lock resource
func (s *FlightBookingServer) RecordTransaction(txID, primaryKey, status string) {
	_, err := s.db.Exec("INSERT INTO transactions (tx_id, primary_key, status) VALUES ($1, $2, $3) ON CONFLICT (tx_id, primary_key) DO UPDATE SET status = $3", txID, primaryKey, status)
	if err != nil {
		log.Printf("Failed to record transaction: %v", err)
	}
}

// 查詢可用座位
func (s *FlightBookingServer) GetSeats(ctx context.Context, req *airlinepb.GetSeatsRequest) (*airlinepb.GetSeatsResponse, error) {
	log.Printf("Fetching available seats for flight %s", req.FlightId)

	var availableSeats int32
	err := s.db.QueryRow("SELECT available_seat FROM flights WHERE flight_id = $1", req.FlightId).Scan(&availableSeats)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Flight %s not found", req.FlightId)
			return &airlinepb.GetSeatsResponse{AvailableSeats: 0, Message: "Flight not found"}, nil
		}
		log.Printf("Database error: %v", err)
		return nil, err
	}

	return &airlinepb.GetSeatsResponse{
		AvailableSeats: availableSeats,
		Message:        "Available seats retrieved",
	}, nil
}

// Propose phase
func (s *FlightBookingServer) ProposeBookSeats(ctx context.Context, req *airlinepb.BookSeatsRequest) (*airlinepb.BookSeatsResponse, error) {
	log.Printf("[Propose] Booking %d seats for flight %s for txId %s", req.SeatCount, req.FlightId, req.TxId)

	if s.IsLocked(req.FlightId) {
		return nil, fmt.Errorf("resource %s is already locked", req.FlightId)
	}
	s.RecordTransaction(req.TxId, req.FlightId, "prepared")

	return &airlinepb.BookSeatsResponse{Message: "Resource locked successfully"}, nil
}

// Commit phase
func (s *FlightBookingServer) CommitBookSeats(ctx context.Context, req *airlinepb.BookSeatsRequest) (*airlinepb.BookSeatsResponse, error) {
	log.Printf("[Commit] Booking %d seats for flight %s for txId %s", req.SeatCount, req.FlightId, req.TxId)

	s.RecordTransaction(req.TxId, req.FlightId, "committed")

	// 開啟交易
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 更新座位數量
	_, err = tx.Exec("UPDATE flights SET available_seat = available_seat - $1 WHERE flight_id = $2", req.SeatCount, req.FlightId)
	if err != nil {
		log.Printf("Failed to update seats: %v", err)
		return nil, err
	}

	// 提交交易
	err = tx.Commit()
	if err != nil {
		log.Printf("Transaction commit failed: %v", err)
		return nil, err
	}

	return &airlinepb.BookSeatsResponse{Message: "Seats booked successfully"}, nil
}

// Cancel phase
func (s *FlightBookingServer) CancelBookSeats(ctx context.Context, req *airlinepb.BookSeatsRequest) (*airlinepb.BookSeatsResponse, error) {
	log.Printf("[Cancel] Booking %d seats for flight %s for txId %s", req.SeatCount, req.FlightId, req.TxId)

	s.RecordTransaction(req.TxId, req.FlightId, "cancelled")
	return &airlinepb.BookSeatsResponse{Message: "Seats booked successfully"}, nil
}

func main() {
	connStr := "host=localhost port=5432 user=postgres dbname=airline2_no_2PC sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 初始化 flights table
	if err := createFlightsTable(db); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Println("Flights Table initialized successfully!")

	// 初始化 transactions table
	if err := createTxTable(db); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Println("Transactions Table initialized successfully!")

	listener, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	airlinepb.RegisterFlightBookingServer(grpcServer, NewFlightBookingServer(db))

	log.Printf("Server listening at %v", listener.Addr())
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
