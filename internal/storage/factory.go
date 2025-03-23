package storage

import (
	"fmt"
	"strconv"

	"github.com/accursedgalaxy/insider-monitor/internal/monitor"
)

// StorageInterface defines the interface for storage implementations
type StorageInterface interface {
	SaveWalletData(data map[string]*monitor.WalletData) error
	LoadWalletData() (map[string]*monitor.WalletData, error)
	IsDataValid() bool
	BackupCurrentData() error
}

// Config represents database configuration
type Config struct {
	Enabled  bool   `json:"enabled"`
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
}

// NewStorage creates a new storage instance based on the provided configuration
func NewStorage(dataDir string, dbConfig *Config) (StorageInterface, error) {
	// If database is not enabled or config is nil, use file storage
	if dbConfig == nil || !dbConfig.Enabled {
		return New(dataDir), nil
	}

	// Choose storage implementation based on type
	switch dbConfig.Type {
	case "postgres":
		return NewPostgresStorage(
			dbConfig.Host,
			strconv.Itoa(dbConfig.Port),
			dbConfig.User,
			dbConfig.Password,
			dbConfig.DBName,
		)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbConfig.Type)
	}
} 