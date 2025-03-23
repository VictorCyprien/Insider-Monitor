package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/accursedgalaxy/insider-monitor/internal/monitor"
	_ "github.com/lib/pq" // PostgreSQL driver
)

type PostgresStorage struct {
	db *sql.DB
}

// NewPostgresStorage creates a new PostgreSQL storage
func NewPostgresStorage(host, port, user, password, dbname string) (*PostgresStorage, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	
	// Initialize database tables
	if err := initDB(db); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	
	return &PostgresStorage{db: db}, nil
}

// initDB creates the necessary tables if they don't exist
func initDB(db *sql.DB) error {
	// Create wallets table
	walletTable := `
	CREATE TABLE IF NOT EXISTS wallets (
		wallet_address TEXT PRIMARY KEY,
		last_scanned TIMESTAMP WITH TIME ZONE
	);`
	
	// Create token_accounts table
	tokenTable := `
	CREATE TABLE IF NOT EXISTS token_accounts (
		id SERIAL PRIMARY KEY,
		wallet_address TEXT REFERENCES wallets(wallet_address) ON DELETE CASCADE,
		token_mint TEXT NOT NULL,
		balance BIGINT NOT NULL,
		last_updated TIMESTAMP WITH TIME ZONE,
		symbol TEXT,
		decimals INTEGER,
		UNIQUE(wallet_address, token_mint)
	);`
	
	if _, err := db.Exec(walletTable); err != nil {
		return fmt.Errorf("failed to create wallets table: %w", err)
	}
	
	if _, err := db.Exec(tokenTable); err != nil {
		return fmt.Errorf("failed to create token_accounts table: %w", err)
	}
	
	return nil
}

// Close the database connection
func (p *PostgresStorage) Close() error {
	return p.db.Close()
}

// SaveWalletData saves wallet data to the PostgreSQL database
func (p *PostgresStorage) SaveWalletData(data map[string]*monitor.WalletData) error {
	// Begin transaction
	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	// Ensure transaction is rolled back if it fails
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	
	// Prepare statements
	upsertWallet, err := tx.Prepare(`
		INSERT INTO wallets (wallet_address, last_scanned)
		VALUES ($1, $2)
		ON CONFLICT (wallet_address) 
		DO UPDATE SET last_scanned = $2;
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare wallet statement: %w", err)
	}
	defer upsertWallet.Close()
	
	upsertToken, err := tx.Prepare(`
		INSERT INTO token_accounts (wallet_address, token_mint, balance, last_updated, symbol, decimals)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (wallet_address, token_mint) 
		DO UPDATE SET balance = $3, last_updated = $4, symbol = $5, decimals = $6;
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare token statement: %w", err)
	}
	defer upsertToken.Close()
	
	// Process each wallet
	for walletAddr, walletData := range data {
		// Insert/update wallet entry
		_, err = upsertWallet.Exec(walletAddr, walletData.LastScanned)
		if err != nil {
			return fmt.Errorf("failed to insert wallet %s: %w", walletAddr, err)
		}
		
		// Process token accounts for this wallet
		for tokenMint, tokenInfo := range walletData.TokenAccounts {
			_, err = upsertToken.Exec(
				walletAddr,
				tokenMint,
				tokenInfo.Balance,
				tokenInfo.LastUpdated,
				tokenInfo.Symbol,
				tokenInfo.Decimals,
			)
			if err != nil {
				return fmt.Errorf("failed to insert token %s for wallet %s: %w", tokenMint, walletAddr, err)
			}
		}
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// LoadWalletData loads wallet data from the PostgreSQL database
func (p *PostgresStorage) LoadWalletData() (map[string]*monitor.WalletData, error) {
	result := make(map[string]*monitor.WalletData)
	
	// Get all wallets
	rows, err := p.db.Query(`SELECT wallet_address, last_scanned FROM wallets`)
	if err != nil {
		return nil, fmt.Errorf("failed to query wallets: %w", err)
	}
	defer rows.Close()
	
	// Process wallet rows
	for rows.Next() {
		var walletAddr string
		var lastScanned time.Time
		
		if err := rows.Scan(&walletAddr, &lastScanned); err != nil {
			return nil, fmt.Errorf("failed to scan wallet row: %w", err)
		}
		
		// Create wallet data entry
		wallet := &monitor.WalletData{
			WalletAddress: walletAddr,
			TokenAccounts: make(map[string]monitor.TokenAccountInfo),
			LastScanned:   lastScanned,
		}
		
		result[walletAddr] = wallet
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating wallet rows: %w", err)
	}
	
	// Get all token accounts
	tokenRows, err := p.db.Query(`
		SELECT wallet_address, token_mint, balance, last_updated, symbol, decimals 
		FROM token_accounts
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query token accounts: %w", err)
	}
	defer tokenRows.Close()
	
	// Process token rows
	for tokenRows.Next() {
		var walletAddr, tokenMint, symbol string
		var balance uint64
		var lastUpdated time.Time
		var decimals uint8
		
		if err := tokenRows.Scan(&walletAddr, &tokenMint, &balance, &lastUpdated, &symbol, &decimals); err != nil {
			return nil, fmt.Errorf("failed to scan token row: %w", err)
		}
		
		// Add token to wallet data
		if wallet, ok := result[walletAddr]; ok {
			wallet.TokenAccounts[tokenMint] = monitor.TokenAccountInfo{
				Balance:     balance,
				LastUpdated: lastUpdated,
				Symbol:      symbol,
				Decimals:    decimals,
			}
		}
	}
	
	if err := tokenRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating token rows: %w", err)
	}
	
	return result, nil
}

// IsDataValid checks if there is valid data in the database
func (p *PostgresStorage) IsDataValid() bool {
	var count int
	err := p.db.QueryRow("SELECT COUNT(*) FROM wallets").Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

// BackupCurrentData creates a JSON backup of the current data
func (p *PostgresStorage) BackupCurrentData() error {
	// Load current data
	data, err := p.LoadWalletData()
	if err != nil {
		return err
	}
	
	// Marshal to JSON and save to a file in the backup directory
	backupDir := "data/backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}
	
	backupPath := filepath.Join(backupDir, fmt.Sprintf("wallet_data_backup_%d.json", time.Now().Unix()))
	file, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	
	return os.WriteFile(backupPath, file, 0644)
} 