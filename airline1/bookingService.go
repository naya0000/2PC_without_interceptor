package main

import (
	"context"
	"database/sql"
	"errors"
	"net"

	_ "github.com/lib/pq"
	airlinepb "github.com/naya0000/2PC_without_interceptor/proto/airlineBooking"
	log "github.com/sirupsen/logrus"
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

func NewFlightBookingServer(db *sql.DB) *FlightBookingServer {
	return &FlightBookingServer{db: db}
}

func createTxTable(db *sql.DB) error {
	db.Exec("DROP TABLE IF EXISTS transactions")
	createTableQuery := `
	CREATE TABLE transactions (
		tx_id VARCHAR(255),
		primary_key VARCHAR(255),
		status VARCHAR(50),
		PRIMARY KEY (primary_key)
	);`
	_, err := db.Exec(createTableQuery)
	return err
}

func createFlightsTable(db *sql.DB) error {
	db.Exec(`DROP TABLE IF EXISTS flights;`)
	createTableQuery := `
	CREATE TABLE flights (
	    flight_id VARCHAR(255) PRIMARY KEY,
	    available_seat INT
	);`
	_, err := db.Exec(createTableQuery)
	if err != nil {
		return err
	}

	// 插入測試數據（如果表是空的）
	insertDataQuery := `
	INSERT INTO flights (flight_id, available_seat) 
	SELECT 'flight1', 2000 WHERE NOT EXISTS (SELECT 1 FROM flights WHERE flight_id = 'flight1');
	`
	_, err = db.Exec(insertDataQuery)
	return err
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
	tx, err := s.db.Begin()
	if err != nil {
		log.Errorf("Failed to begin transaction: %v", err)
		return nil, err
	}

	// var existingStatus string

	// // 檢查 PrimaryKey 是否被其他 tx 鎖定
	// err = tx.QueryRow("SELECT status FROM transactions WHERE primary_key = $1 FOR UPDATE", req.FlightId).Scan(&existingStatus)
	// if err != nil && err != sql.ErrNoRows {
	// 	tx.Rollback()
	// 	return nil, errors.New("failed to check existing transaction")
	// }

	// if existingStatus == "prepared" {
	// 	tx.Rollback()
	// 	return nil, errors.New("resource is locked")
	// }

	// _, err = tx.Exec("INSERT INTO transactions (tx_id, primary_key, status) VALUES ($1, $2, $3) ON CONFLICT (primary_key) DO UPDATE SET tx_id = $1, status = $3", req.TxId, req.FlightId, "prepared")

	var updatedTxID string
	err = tx.QueryRow(`
		INSERT INTO transactions (tx_id, primary_key, status)
		VALUES ($1, $2, 'prepared')
		ON CONFLICT (primary_key) DO UPDATE
		SET tx_id = EXCLUDED.tx_id, status = 'prepared'
		WHERE transactions.status IS DISTINCT FROM 'prepared'
		RETURNING tx_id
	`, req.TxId, req.FlightId).Scan(&updatedTxID)

	if err == sql.ErrNoRows {
		tx.Rollback()
		log.Errorf("Resource is locked")
		return nil, errors.New("resource is locked")
	}

	if err != nil {
		tx.Rollback()
		log.Errorf("Propose insert/update failed: %v", err)
		return nil, errors.New("failed to propose transaction")
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return nil, errors.New("failed to commit transaction")
	}

	return &airlinepb.BookSeatsResponse{Message: "Seats prepared successfully"}, nil
}

// Commit phase
func (s *FlightBookingServer) CommitBookSeats(ctx context.Context, req *airlinepb.BookSeatsRequest) (*airlinepb.BookSeatsResponse, error) {
	log.Printf("[Commit] Booking %d seats for flight %s for txId %s", req.SeatCount, req.FlightId, req.TxId)

	tx, err := s.db.Begin()
	if err != nil {
		log.Errorf("Failed to begin transaction: %v", err)
		return nil, errors.New("failed to begin transaction")
	}
	var availableSeats int32
	err = tx.QueryRow("SELECT available_seat FROM flights WHERE flight_id = $1 FOR UPDATE", req.FlightId).Scan(&availableSeats)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Flight %s not found", req.FlightId)
			return &airlinepb.BookSeatsResponse{Message: "Flight not found"}, nil
		}
		log.Printf("Database error: %v", err)
		return nil, err
	}

	newAvailableSeats := availableSeats - req.SeatCount
	if newAvailableSeats < 0 {
		log.Printf("Not enough seats available for flight %s", req.FlightId)
		return &airlinepb.BookSeatsResponse{Message: "Not enough seats available"}, nil
	}

	// 更新座位數量
	_, err = tx.Exec("UPDATE flights SET available_seat = available_seat - $1 WHERE flight_id = $2", req.SeatCount, req.FlightId)
	if err != nil {
		log.Printf("Failed to update seats: %v", err)
		return nil, err
	}

	// 更新 tx 狀態
	var selectedTxId string
	err = tx.QueryRow("SELECT tx_id FROM transactions WHERE primary_key = $1 AND status =$2 FOR UPDATE", req.FlightId, "prepared").Scan(&selectedTxId)
	if err != nil {
		log.Errorf("Failed to select transaction: %v", err)
		tx.Rollback()
		return nil, errors.New("failed to select transaction")
	}

	if selectedTxId != req.TxId {
		tx.Rollback()
		return nil, errors.New("transaction is not the owner of the resource")
	}

	_, err = tx.Exec("UPDATE transactions SET status = $1 WHERE tx_id = $2 AND primary_key = $3", "committed", req.TxId, req.FlightId)
	if err != nil {
		log.Errorf("Failed to update transaction: %v", err)
		tx.Rollback()
		return nil, errors.New("failed to update transaction")
	}

	if err = tx.Commit(); err != nil {
		log.Errorf("Transaction commit failed: %v", err)
		tx.Rollback()
		return nil, errors.New("failed to commit transaction")
	}

	return &airlinepb.BookSeatsResponse{Message: "Seats booked successfully", AvailableSeats: newAvailableSeats}, nil
}

// Cancel phase
func (s *FlightBookingServer) CancelBookSeats(ctx context.Context, req *airlinepb.BookSeatsRequest) (*airlinepb.BookSeatsResponse, error) {
	tx, err := s.db.Begin()
	if err != nil {
		log.Errorf("Failed to begin transaction: %v", err)
		return nil, errors.New("failed to begin transaction")
	}

	var selectedTxId string
	err = tx.QueryRow("SELECT tx_id FROM transactions WHERE primary_key = $1 FOR UPDATE", req.FlightId).Scan(&selectedTxId)
	if err != nil {
		log.Errorf("Failed to select transaction: %v", err)
		tx.Rollback()
		return nil, errors.New("failed to select transaction")
	}

	if selectedTxId != req.TxId {
		// log.Errorf("Transaction %s is not the owner of the resource", txID)
		tx.Rollback()
		return nil, errors.New("transaction is not the owner of the resource")
	}

	_, err = tx.Exec("DELETE FROM transactions WHERE primary_key = $1", req.FlightId)
	if err != nil {
		log.Errorf("Failed to delete transaction: %v", err)
		tx.Rollback()
		return nil, errors.New("failed to delete transaction")
	}

	if err = tx.Commit(); err != nil {
		log.Errorf("Transaction commit failed: %v", err)
		tx.Rollback()
		return nil, errors.New("failed to commit transaction")
	}

	return &airlinepb.BookSeatsResponse{Message: "Seats booking cancelled successfully"}, nil
}

func main() {
	connStr := "host=localhost port=5432 user=postgres dbname=airline1_no_interceptor sslmode=disable"
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

	listener, err := net.Listen("tcp", ":50051")
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
