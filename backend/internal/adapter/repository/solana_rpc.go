package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/rs/zerolog"
)

type SolanaRPC struct {
	client *rpc.Client
	logger *zerolog.Logger
}

func NewSolanaRPC(rpcURL string, logger *zerolog.Logger) *SolanaRPC {
	return &SolanaRPC{
		client: rpc.New(rpcURL),
		logger: logger,
	}
}

func (r *SolanaRPC) GetAccountInfo(ctx context.Context, address string) ([]byte, error) {
	pk, err := solana.PublicKeyFromBase58(address)
	if err != nil {
		return nil, fmt.Errorf("solana rpc parse address: %w", err)
	}

	resp, err := r.client.GetAccountInfo(ctx, pk)
	if err != nil {
		return nil, fmt.Errorf("solana rpc getAccountInfo: %w", err)
	}
	if resp == nil || resp.Value == nil {
		return nil, fmt.Errorf("solana rpc getAccountInfo: account not found")
	}

	return resp.Value.Data.GetBinary(), nil
}

func (r *SolanaRPC) SendTransaction(ctx context.Context, txBytes []byte) (string, error) {
	tx, err := solana.TransactionFromBytes(txBytes)
	if err != nil {
		return "", fmt.Errorf("solana rpc parse transaction: %w", err)
	}

	sig, err := r.client.SendTransaction(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("solana rpc sendTransaction: %w", err)
	}

	return sig.String(), nil
}

func (r *SolanaRPC) SimulateTransaction(ctx context.Context, txBytes []byte) error {
	tx, err := solana.TransactionFromBytes(txBytes)
	if err != nil {
		return fmt.Errorf("solana rpc parse transaction: %w", err)
	}

	resp, err := r.client.SimulateTransaction(ctx, tx)
	if err != nil {
		return fmt.Errorf("solana rpc simulateTransaction: %w", err)
	}
	if resp.Value.Err != nil {
		return fmt.Errorf("solana rpc simulation failed: %v", resp.Value.Err)
	}

	return nil
}

// LoadKeypair reads a Solana keypair JSON file (array of 64 bytes) and returns the private key.
func LoadKeypair(path string) (solana.PrivateKey, error) {
	path = strings.Replace(path, "~", os.Getenv("HOME"), 1)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading keypair file %q: %w", path, err)
	}

	var keyBytes []byte
	if err := json.Unmarshal(data, &keyBytes); err != nil {
		return nil, fmt.Errorf("parsing keypair JSON: %w", err)
	}

	return solana.PrivateKey(keyBytes), nil
}

func (r *SolanaRPC) GetSignaturesForAddress(ctx context.Context, address string, limit int) ([]string, error) {
	pk, err := solana.PublicKeyFromBase58(address)
	if err != nil {
		return nil, fmt.Errorf("solana rpc parse address: %w", err)
	}

	resp, err := r.client.GetSignaturesForAddressWithOpts(ctx, pk, &rpc.GetSignaturesForAddressOpts{
		Limit: &limit,
	})
	if err != nil {
		return nil, fmt.Errorf("solana rpc getSignaturesForAddress: %w", err)
	}

	sigs := make([]string, len(resp))
	for i, s := range resp {
		sigs[i] = s.Signature.String()
	}
	return sigs, nil
}
