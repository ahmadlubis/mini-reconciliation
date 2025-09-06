package usecase

import (
	"context"
	"mini-reconciliation/internal/domain"
)

// TransactionRepository defines the interface for fetching transaction data.
// The usecase layer depends on this interface, not on a concrete implementation.
//
//go:generate mockgen -destination=mocks/mock_repository.go -source=interface.go TransactionRepository
type TransactionRepository interface {
	GetSystemTransactions(ctx context.Context, path string) ([]domain.SystemTransaction, error)
	GetBankTransactions(ctx context.Context, paths []string) ([]domain.BankTransaction, error)
}
