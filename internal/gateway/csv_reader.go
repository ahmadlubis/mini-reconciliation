package gateway

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"mini-reconciliation/internal/domain"
)

// CSVTransactionRepository implements the TransactionRepository interface for CSV files.
type CSVTransactionRepository struct{}

// NewCSVTransactionRepository creates a new repository instance.
func NewCSVTransactionRepository() *CSVTransactionRepository {
	return &CSVTransactionRepository{}
}

// GetSystemTransactions reads and parses the system transactions CSV file.
func (r *CSVTransactionRepository) GetSystemTransactions(ctx context.Context, path string) ([]domain.SystemTransaction, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open system transaction file %s: %w", path, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Skip header
	if _, err := reader.Read(); err != nil {
		return nil, fmt.Errorf("failed to read header from %s: %w", path, err)
	}

	var transactions []domain.SystemTransaction
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading record from %s: %w", path, err)
		}

		amount, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse amount '%s': %w", record[1], err)
		}

		txTime, err := time.Parse(time.RFC3339, record[3])
		if err != nil {
			return nil, fmt.Errorf("could not parse transactionTime '%s': %w", record[3], err)
		}

		tx := domain.SystemTransaction{
			TrxID:           record[0],
			Amount:          amount,
			Type:            domain.TransactionType(record[2]),
			TransactionTime: txTime,
		}
		transactions = append(transactions, tx)
	}
	return transactions, nil
}

// GetBankTransactions reads and parses multiple bank statement CSV files.
func (r *CSVTransactionRepository) GetBankTransactions(ctx context.Context, paths []string) ([]domain.BankTransaction, error) {
	var allTransactions []domain.BankTransaction

	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open bank statement file %s: %w", path, err)
		}
		defer file.Close()

		reader := csv.NewReader(file)
		// Skip header
		if _, err := reader.Read(); err != nil {
			return nil, fmt.Errorf("failed to read header from %s: %w", path, err)
		}

		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("error reading record from %s: %w", path, err)
			}

			amount, err := strconv.ParseFloat(record[1], 64)
			if err != nil {
				return nil, fmt.Errorf("could not parse amount '%s': %w", record[1], err)
			}

			date, err := time.Parse("2006-01-02", record[2])
			if err != nil {
				return nil, fmt.Errorf("could not parse date '%s': %w", record[2], err)
			}

			tx := domain.BankTransaction{
				UniqueIdentifier: record[0],
				Amount:           amount,
				Date:             date,
				Description:      record[3],
				BankSource:       filepath.Base(path),
			}

			// Normalize the transaction for easier matching
			if tx.Amount < 0 {
				tx.Type = domain.TransactionTypeDebit
				tx.NormalizedAmount = math.Abs(tx.Amount)
			} else {
				tx.Type = domain.TransactionTypeCredit
				tx.NormalizedAmount = tx.Amount
			}

			allTransactions = append(allTransactions, tx)
		}
	}
	return allTransactions, nil
}
