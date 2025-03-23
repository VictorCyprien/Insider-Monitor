package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

// WalletData represents the structure of wallet_data.json
type WalletData map[string]WalletInfo

// WalletInfo holds information about a wallet
type WalletInfo struct {
	WalletAddress string                    `json:"wallet_address"`
	TokenAccounts map[string]TokenAccount   `json:"token_accounts"`
	LastScanned   string                    `json:"last_scanned"`
}

// TokenAccount represents a token account for a wallet
type TokenAccount struct {
	Balance      int64       `json:"balance"`
	DecimalValue json.Number `json:"decimal_value,omitempty"`
	LastUpdated  string      `json:"last_updated"`
	Symbol       string      `json:"symbol"`
	Decimals     int         `json:"decimals"`
}

var walletData WalletData

func main() {
	// Load the wallet data from the JSON file
	if err := loadWalletData(); err != nil {
		log.Fatalf("Failed to load wallet data: %v", err)
	}

	// Create a new router
	r := mux.NewRouter()

	// Define API endpoints
	r.HandleFunc("/api/wallets", getAllWallets).Methods("GET")
	r.HandleFunc("/api/wallets/{address}", getWallet).Methods("GET")
	r.HandleFunc("/api/wallets/{address}/tokens", getWalletTokens).Methods("GET")
	r.HandleFunc("/api/wallets/{address}/tokens/{token}", getWalletToken).Methods("GET")

	// Start the server
	port := getEnv("API_PORT", "8080")
	fmt.Printf("Starting API server on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

// loadWalletData loads the wallet data from the JSON file
func loadWalletData() error {
	// Get the path to the wallet_data.json file
	dataDir := getEnv("DATA_DIR", "data")
	filePath := filepath.Join(dataDir, "wallet_data.json")

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read wallet data file: %w", err)
	}

	// Unmarshal the JSON data
	if err := json.Unmarshal(data, &walletData); err != nil {
		return fmt.Errorf("failed to unmarshal wallet data: %w", err)
	}

	// Calculate decimal values for all tokens
	for walletAddress, wallet := range walletData {
		for tokenAddress, token := range wallet.TokenAccounts {
			decimalValue := calculateDecimalValue(token.Balance, token.Decimals)
			token.DecimalValue = json.Number(decimalValue)
			wallet.TokenAccounts[tokenAddress] = token
		}
		walletData[walletAddress] = wallet
	}

	return nil
}

// calculateDecimalValue converts the raw balance to a decimal value based on decimals
func calculateDecimalValue(balance int64, decimals int) string {
	if decimals <= 0 {
		return fmt.Sprintf("%d", balance)
	}

	// Use big.Float for precise decimal calculation
	balanceFloat := new(big.Float).SetInt64(balance)
	divisor := new(big.Float).SetFloat64(math.Pow10(decimals))
	result := new(big.Float).Quo(balanceFloat, divisor)

	// Format to string with appropriate precision
	return result.Text('f', decimals)
}

// getAllWallets returns all wallet data
func getAllWallets(w http.ResponseWriter, r *http.Request) {
	// Check if the client wants raw values or with decimal conversion
	includeDecimal := r.URL.Query().Get("decimal") != "false"
	
	if !includeDecimal {
		// Return raw data without decimal values
		for walletAddress, wallet := range walletData {
			for tokenAddress, token := range wallet.TokenAccounts {
				token.DecimalValue = ""
				wallet.TokenAccounts[tokenAddress] = token
			}
			walletData[walletAddress] = wallet
		}
	}
	
	respondWithJSON(w, http.StatusOK, walletData)
}

// getWallet returns data for a specific wallet
func getWallet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	wallet, ok := walletData[address]
	if !ok {
		respondWithError(w, http.StatusNotFound, "Wallet not found")
		return
	}

	// Check if the client wants raw values or with decimal conversion
	includeDecimal := r.URL.Query().Get("decimal") != "false"
	
	if !includeDecimal {
		// Return raw data without decimal values
		for tokenAddress, token := range wallet.TokenAccounts {
			token.DecimalValue = ""
			wallet.TokenAccounts[tokenAddress] = token
		}
	}

	respondWithJSON(w, http.StatusOK, wallet)
}

// getWalletTokens returns all token data for a specific wallet
func getWalletTokens(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	wallet, ok := walletData[address]
	if !ok {
		respondWithError(w, http.StatusNotFound, "Wallet not found")
		return
	}

	// Check if the client wants raw values or with decimal conversion
	includeDecimal := r.URL.Query().Get("decimal") != "false"
	
	if !includeDecimal {
		// Return raw data without decimal values
		for tokenAddress, token := range wallet.TokenAccounts {
			token.DecimalValue = ""
			wallet.TokenAccounts[tokenAddress] = token
		}
	}

	respondWithJSON(w, http.StatusOK, wallet.TokenAccounts)
}

// getWalletToken returns data for a specific token in a wallet
func getWalletToken(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]
	tokenAddress := vars["token"]

	wallet, ok := walletData[address]
	if !ok {
		respondWithError(w, http.StatusNotFound, "Wallet not found")
		return
	}

	token, ok := wallet.TokenAccounts[tokenAddress]
	if !ok {
		respondWithError(w, http.StatusNotFound, "Token not found")
		return
	}

	// Check if the client wants raw values or with decimal conversion
	includeDecimal := r.URL.Query().Get("decimal") != "false"
	
	if !includeDecimal {
		// Return raw data without decimal values
		token.DecimalValue = ""
	}

	respondWithJSON(w, http.StatusOK, token)
}

// respondWithError returns an error response
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

// respondWithJSON returns a JSON response
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	
	if payload != nil {
		response, err := json.Marshal(payload)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Failed to marshal response"}`))
			return
		}
		w.Write(response)
	}
}

// getEnv returns the value of an environment variable or a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return strings.TrimSpace(value)
} 