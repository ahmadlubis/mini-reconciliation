package gateway

import (
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mini-reconciliation/internal/domain"

	"github.com/stretchr/testify/assert"
)

func TestCSVTransactionRepository_GetSystemTransactions(t *testing.T) {
	tests := []struct {
		name     string
		csvData  [][]string
		expected []domain.SystemTransaction
		wantErr  bool
	}{
		{
			name: "valid system transactions",
			csvData: [][]string{
				{"trxID", "amount", "type", "transactionTime"},
				{"SYS001", "150.00", "DEBIT", "2025-09-01T10:00:00Z"},
				{"SYS002", "200.50", "CREDIT", "2025-09-01T11:30:00Z"},
				{"SYS003", "75.00", "DEBIT", "2025-09-02T09:00:00Z"},
			},
			expected: []domain.SystemTransaction{
				{
					TrxID:           "SYS001",
					Amount:          150.00,
					Type:            domain.TransactionType("DEBIT"),
					TransactionTime: mustParseTime("2025-09-01T10:00:00Z"),
				},
				{
					TrxID:           "SYS002",
					Amount:          200.50,
					Type:            domain.TransactionType("CREDIT"),
					TransactionTime: mustParseTime("2025-09-01T11:30:00Z"),
				},
				{
					TrxID:           "SYS003",
					Amount:          75.00,
					Type:            domain.TransactionType("DEBIT"),
					TransactionTime: mustParseTime("2025-09-02T09:00:00Z"),
				},
			},
			wantErr: false,
		},
		{
			name: "empty file with header only",
			csvData: [][]string{
				{"trxID", "amount", "type", "transactionTime"},
			},
			expected: nil,
			wantErr:  false,
		},
		{
			name: "invalid amount format",
			csvData: [][]string{
				{"trxID", "amount", "type", "transactionTime"},
				{"SYS001", "invalid_amount", "DEBIT", "2025-09-01T10:00:00Z"},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "invalid time format",
			csvData: [][]string{
				{"trxID", "amount", "type", "transactionTime"},
				{"SYS001", "150.00", "DEBIT", "invalid_time"},
			},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary CSV file
			tmpFile, err := createTempCSV(tt.csvData)
			if err != nil {
				t.Fatalf("Failed to create temp CSV file: %v", err)
			}
			defer os.Remove(tmpFile)

			repo := NewCSVTransactionRepository()
			ctx := context.Background()

			got, err := repo.GetSystemTransactions(ctx, tmpFile)
			if tt.wantErr {
				assert.Error(t, err, "Expected error but got nil")
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestCSVTransactionRepository_GetSystemTransactions_FileErrors(t *testing.T) {
	repo := NewCSVTransactionRepository()
	ctx := context.Background()

	t.Run("file not found", func(t *testing.T) {
		_, err := repo.GetSystemTransactions(ctx, "nonexistent_file.csv")
		if err == nil {
			t.Error("Expected error for nonexistent file, got nil")
		}
	})

	t.Run("file with no header", func(t *testing.T) {
		// Create empty file
		tmpFile, err := os.CreateTemp("", "empty_*.csv")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		_, err = repo.GetSystemTransactions(ctx, tmpFile.Name())
		if err == nil {
			t.Error("Expected error for empty file, got nil")
		}
	})
}

func TestCSVTransactionRepository_GetBankTransactions(t *testing.T) {
	tests := []struct {
		name      string
		filesData [][]string // Each element represents a CSV file
		expected  []domain.BankTransaction
		wantErr   bool
	}{
		{
			name: "valid bank transactions from single file",
			filesData: [][]string{
				{
					"unique_identifier,amount,date,description",
					"BANK_A_1,-150.00,2025-09-01,Payment for INV001 trxID:SYS001",
					"BANK_A_2,200.50,2025-09-01,Incoming Transfer",
					"BANK_A_3,-75.00,2025-09-02,Withdrawal ATM Central",
				},
			},
			expected: []domain.BankTransaction{
				{
					UniqueIdentifier: "BANK_A_1",
					Amount:           -150.00,
					Date:             mustParseDate("2025-09-01"),
					Description:      "Payment for INV001 trxID:SYS001",
					BankSource:       "test_bank_0.csv",
					Type:             domain.TransactionTypeDebit,
					NormalizedAmount: 150.00,
				},
				{
					UniqueIdentifier: "BANK_A_2",
					Amount:           200.50,
					Date:             mustParseDate("2025-09-01"),
					Description:      "Incoming Transfer",
					BankSource:       "test_bank_0.csv",
					Type:             domain.TransactionTypeCredit,
					NormalizedAmount: 200.50,
				},
				{
					UniqueIdentifier: "BANK_A_3",
					Amount:           -75.00,
					Date:             mustParseDate("2025-09-02"),
					Description:      "Withdrawal ATM Central",
					BankSource:       "test_bank_0.csv",
					Type:             domain.TransactionTypeDebit,
					NormalizedAmount: 75.00,
				},
			},
			wantErr: false,
		},
		{
			name: "valid bank transactions from multiple files",
			filesData: [][]string{
				{
					"unique_identifier,amount,date,description",
					"BANK_A_1,-150.00,2025-09-01,Payment for INV001",
				},
				{
					"unique_identifier,amount,date,description",
					"BANK_B_1,500.00,2025-09-03,Deposit from Client X",
				},
			},
			expected: []domain.BankTransaction{
				{
					UniqueIdentifier: "BANK_A_1",
					Amount:           -150.00,
					Date:             mustParseDate("2025-09-01"),
					Description:      "Payment for INV001",
					BankSource:       "test_bank_0.csv",
					Type:             domain.TransactionTypeDebit,
					NormalizedAmount: 150.00,
				},
				{
					UniqueIdentifier: "BANK_B_1",
					Amount:           500.00,
					Date:             mustParseDate("2025-09-03"),
					Description:      "Deposit from Client X",
					BankSource:       "test_bank_1.csv",
					Type:             domain.TransactionTypeCredit,
					NormalizedAmount: 500.00,
				},
			},
			wantErr: false,
		},
		{
			name: "empty files with headers only",
			filesData: [][]string{
				{"unique_identifier,amount,date,description"},
				{"unique_identifier,amount,date,description"},
			},
			expected: []domain.BankTransaction{},
			wantErr:  false,
		},
		{
			name: "invalid amount format",
			filesData: [][]string{
				{
					"unique_identifier,amount,date,description",
					"BANK_A_1,invalid_amount,2025-09-01,Payment",
				},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "invalid date format",
			filesData: [][]string{
				{
					"unique_identifier,amount,date,description",
					"BANK_A_1,-150.00,invalid_date,Payment",
				},
			},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary CSV files
			var tmpFiles []string
			for i, fileData := range tt.filesData {
				tmpFile, err := createTempCSVFromLines(fileData, "test_bank_"+string(rune('0'+i))+".csv")
				if err != nil {
					t.Fatalf("Failed to create temp CSV file %d: %v", i, err)
				}
				tmpFiles = append(tmpFiles, tmpFile)
			}

			// Clean up files after test
			defer func() {
				for _, file := range tmpFiles {
					os.Remove(file)
				}
			}()

			repo := NewCSVTransactionRepository()
			ctx := context.Background()

			got, err := repo.GetBankTransactions(ctx, tmpFiles)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetBankTransactions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(got) != len(tt.expected) {
					t.Errorf("GetBankTransactions() got %d transactions, want %d", len(got), len(tt.expected))
					return
				}

				for i, expectedTx := range tt.expected {
					if i >= len(got) {
						t.Errorf("Missing transaction at index %d", i)
						continue
					}

					gotTx := got[i]
					if !compareBankTransactions(gotTx, expectedTx) {
						t.Errorf("GetBankTransactions() transaction[%d] = %+v, want %+v", i, gotTx, expectedTx)
					}
				}
			}
		})
	}
}

func TestCSVTransactionRepository_GetBankTransactions_FileErrors(t *testing.T) {
	repo := NewCSVTransactionRepository()
	ctx := context.Background()

	t.Run("file not found", func(t *testing.T) {
		_, err := repo.GetBankTransactions(ctx, []string{"nonexistent_file.csv"})
		if err == nil {
			t.Error("Expected error for nonexistent file, got nil")
		}
	})

	t.Run("one valid file and one invalid file", func(t *testing.T) {
		// Create one valid temp file
		validFile, err := createTempCSVFromLines([]string{
			"unique_identifier,amount,date,description",
			"BANK_A_1,-150.00,2025-09-01,Payment",
		}, "valid.csv")
		if err != nil {
			t.Fatalf("Failed to create valid temp file: %v", err)
		}
		defer os.Remove(validFile)

		_, err = repo.GetBankTransactions(ctx, []string{validFile, "nonexistent.csv"})
		if err == nil {
			t.Error("Expected error when one file doesn't exist, got nil")
		}
	})
}

// Helper functions

func createTempCSV(data [][]string) (string, error) {
	tmpFile, err := os.CreateTemp("", "test_*.csv")
	if err != nil {
		return "", err
	}

	writer := csv.NewWriter(tmpFile)

	for _, record := range data {
		if err := writer.Write(record); err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return "", err
		}
	}

	// Flush the writer to ensure data is written to the file
	writer.Flush()
	if err := writer.Error(); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", err
	}

	// Close the file
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

func createTempCSVFromLines(lines []string, filename string) (string, error) {
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, filename)

	file, err := os.Create(tmpFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	for i, line := range lines {
		if i > 0 {
			file.WriteString("\n")
		}
		file.WriteString(line)
	}

	return tmpFile, nil
}

func mustParseTime(timeStr string) time.Time {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		panic(err)
	}
	return t
}

func mustParseDate(dateStr string) time.Time {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		panic(err)
	}
	return t
}

func compareBankTransactions(got, want domain.BankTransaction) bool {
	return got.UniqueIdentifier == want.UniqueIdentifier &&
		got.Amount == want.Amount &&
		got.Date.Equal(want.Date) &&
		got.Description == want.Description &&
		got.BankSource == want.BankSource &&
		got.Type == want.Type &&
		got.NormalizedAmount == want.NormalizedAmount
}

// Benchmark tests

func BenchmarkGetSystemTransactions(b *testing.B) {
	// Create a large CSV file for benchmarking
	data := [][]string{{"trxID", "amount", "type", "transactionTime"}}
	for i := 0; i < 1000; i++ {
		data = append(data, []string{
			"SYS" + string(rune('0'+i%10)),
			"150.00",
			"DEBIT",
			"2025-09-01T10:00:00Z",
		})
	}

	tmpFile, err := createTempCSV(data)
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	repo := NewCSVTransactionRepository()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.GetSystemTransactions(ctx, tmpFile)
		if err != nil {
			b.Fatalf("Error in benchmark: %v", err)
		}
	}
}

func BenchmarkGetBankTransactions(b *testing.B) {
	// Create a large CSV file for benchmarking
	lines := []string{"unique_identifier,amount,date,description"}
	for i := 0; i < 1000; i++ {
		lines = append(lines, "BANK_A_1,-150.00,2025-09-01,Payment description")
	}

	tmpFile, err := createTempCSVFromLines(lines, "benchmark.csv")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	repo := NewCSVTransactionRepository()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.GetBankTransactions(ctx, []string{tmpFile})
		if err != nil {
			b.Fatalf("Error in benchmark: %v", err)
		}
	}
}
