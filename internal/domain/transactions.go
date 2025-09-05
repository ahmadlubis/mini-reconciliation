package domain

import "time"

// TransactionType defines the nature of the transaction (DEBIT or CREDIT).
type TransactionType string

const (
	DEBIT  TransactionType = "DEBIT"
	CREDIT TransactionType = "CREDIT"
)

// SystemTransaction represents a transaction from Amartha's internal system.
type SystemTransaction struct {
	TrxID           string          `json:"trxID"`
	Amount          float64         `json:"amount"`
	Type            TransactionType `json:"type"`
	TransactionTime time.Time       `json:"transactionTime"`
}

// BankTransaction represents a transaction from a bank statement.
// It includes additional fields for normalization and tracking.
type BankTransaction struct {
	UniqueIdentifier string    `json:"unique_identifier"`
	Amount           float64   `json:"amount"` // Can be negative
	Date             time.Time `json:"date"`
	Description      string    `json:"description"`
	BankSource       string    `json:"bank_source"` // e.g., "bank_A_statement.csv"

	// Normalized fields for reconciliation logic
	NormalizedAmount float64         `json:"-"`
	Type             TransactionType `json:"-"`
}
