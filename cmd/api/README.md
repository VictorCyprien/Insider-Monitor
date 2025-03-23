# Wallet Data API

This API provides access to wallet data stored in the `data/wallet_data.json` file.

## Running the API Server

```bash
# From the project root
go run cmd/api/main.go
```

By default, the server runs on port 8080. You can change this by setting the `API_PORT` environment variable:

```bash
API_PORT=3000 go run cmd/api/main.go
```

You can also specify the directory where the wallet data file is located using the `DATA_DIR` environment variable (defaults to "data"):

```bash
DATA_DIR=/path/to/data go run cmd/api/main.go
```

## API Endpoints

The API provides the following endpoints:

### Get All Wallets

```
GET /api/wallets
```

Returns all wallet data.

### Get a Specific Wallet

```
GET /api/wallets/{address}
```

Returns data for a specific wallet address.

### Get All Tokens for a Wallet

```
GET /api/wallets/{address}/tokens
```

Returns all token accounts for a specific wallet.

### Get a Specific Token for a Wallet

```
GET /api/wallets/{address}/tokens/{token}
```

Returns data for a specific token in a wallet.

## Token Decimal Representation

By default, the API returns both the raw integer balance and a decimal representation in the `decimal_value` field, which is calculated based on the token's `decimals` field.

For example, if a token has a balance of `2087290570` and 9 decimal places, the `decimal_value` will be `2.087290570`.

To disable the decimal value calculation and only receive raw balances, add the query parameter `decimal=false` to any of the API endpoints:

```
GET /api/wallets?decimal=false
```

## Response Format

All responses are in JSON format. Success responses will return the requested data, while error responses will have the following format:

```json
{
  "error": "Error message"
}
```

## Example Requests

### Get All Wallets

```bash
curl http://localhost:8080/api/wallets
```

### Get a Specific Wallet

```bash
curl http://localhost:8080/api/wallets/8DptPNNcFFMNgaxqQqVYo6ikuHKKEmq1AJt4yUWKRDH6
```

### Get All Tokens for a Wallet

```bash
curl http://localhost:8080/api/wallets/8DptPNNcFFMNgaxqQqVYo6ikuHKKEmq1AJt4yUWKRDH6/tokens
```

### Get a Specific Token for a Wallet

```bash
curl http://localhost:8080/api/wallets/8DptPNNcFFMNgaxqQqVYo6ikuHKKEmq1AJt4yUWKRDH6/tokens/So11111111111111111111111111111111111111112
```

### Get a Wallet Without Decimal Values

```bash
curl "http://localhost:8080/api/wallets/8DptPNNcFFMNgaxqQqVYo6ikuHKKEmq1AJt4yUWKRDH6?decimal=false"
``` 