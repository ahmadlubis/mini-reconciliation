package usecase_test

import (
	"context"
	"errors"
	"mini-reconciliation/internal/domain"
	"mini-reconciliation/internal/usecase"
	mock_usecase "mini-reconciliation/internal/usecase/mocks"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestReconciliationUseCase_Reconcile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	start := baseTime
	end := baseTime.AddDate(0, 0, 7) // 7 days later

	tests := []struct {
		name            string
		systemPath      string
		bankPaths       []string
		start           time.Time
		end             time.Time
		systemTxs       []domain.SystemTransaction
		bankTxs         []domain.BankTransaction
		systemRepoError error
		bankRepoError   error
		want            *domain.ReconciliationReport
		wantErr         bool
	}{
		{
			name:       "successful reconciliation with all matching types",
			systemPath: "/examples/transactions/system_transactions.csv",
			bankPaths:  []string{"/examples/statements/statement_bank_A.csv", "/examples/statements/statement_bank_B.csv"},
			start:      start,
			end:        end,
			systemTxs: []domain.SystemTransaction{
				{
					TrxID:           "TRX001",
					TransactionTime: baseTime.AddDate(0, 0, 1),
					Type:            domain.TransactionTypeDebit,
					Amount:          100.00,
				},
				{
					TrxID:           "TRX002",
					TransactionTime: baseTime.AddDate(0, 0, 2),
					Type:            domain.TransactionTypeCredit,
					Amount:          250.50,
				},
				{
					TrxID:           "TRX003",
					TransactionTime: baseTime.AddDate(0, 0, 3),
					Type:            domain.TransactionTypeDebit,
					Amount:          75.25,
				},
			},
			bankTxs: []domain.BankTransaction{
				{
					UniqueIdentifier: "BANK001",
					Date:             baseTime.AddDate(0, 0, 1),
					Type:             domain.TransactionTypeDebit,
					NormalizedAmount: 100.00,
					Description:      "Purchase trxID:TRX001",
					BankSource:       "Bank1",
				},
				{
					UniqueIdentifier: "BANK002",
					Date:             baseTime.AddDate(0, 0, 2),
					Type:             domain.TransactionTypeCredit,
					NormalizedAmount: 250.50,
					Description:      "Deposit",
					BankSource:       "Bank1",
				},
				{
					UniqueIdentifier: "BANK003",
					Date:             baseTime.AddDate(0, 0, 3),
					Type:             domain.TransactionTypeDebit,
					NormalizedAmount: 75.25,
					Description:      "Withdrawal",
					BankSource:       "Bank2",
				},
			},
			want: &domain.ReconciliationReport{
				ReconciliationSummary: domain.Summary{
					TimeframeStart:                   start.Format(time.DateOnly),
					TimeframeEnd:                     end.Format(time.DateOnly),
					TotalSystemTransactionsProcessed: 3,
					TotalBankTransactionsProcessed:   3,
					MatchedTransactions:              3,
				},
				DiscrepantTransactions: domain.DiscrepantTransactions{
					Details: make([]domain.DiscrepancyDetail, 0),
				},
				UnmatchedTransactions: domain.UnmatchedTransactions{
					BankMissingFromSystem: make(map[string][]domain.BankTransaction),
				},
			},
		},
		{
			name:       "reconciliation with discrepancies",
			systemPath: "/examples/transactions/system_transactions.csv",
			bankPaths:  []string{"/examples/statements/statement_bank_A.csv", "/examples/statements/statement_bank_B.csv"},
			start:      start,
			end:        end,
			systemTxs: []domain.SystemTransaction{
				{
					TrxID:           "TRX001",
					TransactionTime: baseTime.AddDate(0, 0, 1),
					Type:            domain.TransactionTypeDebit,
					Amount:          100.00,
				},
				{
					TrxID:           "TRX002",
					TransactionTime: baseTime.AddDate(0, 0, 2),
					Type:            domain.TransactionTypeCredit,
					Amount:          250.50,
				},
			},
			bankTxs: []domain.BankTransaction{
				{
					UniqueIdentifier: "BANK001",
					Date:             baseTime.AddDate(0, 0, 1),
					Type:             domain.TransactionTypeDebit,
					NormalizedAmount: 99.95, // Small discrepancy
					Description:      "Purchase trxID:TRX001",
					BankSource:       "Bank1",
				},
				{
					UniqueIdentifier: "BANK002",
					Date:             baseTime.AddDate(0, 0, 2),
					Type:             domain.TransactionTypeCredit,
					NormalizedAmount: 251.00, // Discrepancy
					Description:      "Deposit trxID:TRX002",
					BankSource:       "Bank1",
				},
			},
			want: &domain.ReconciliationReport{
				ReconciliationSummary: domain.Summary{
					TimeframeStart:                   start.Format(time.DateOnly),
					TimeframeEnd:                     end.Format(time.DateOnly),
					TotalSystemTransactionsProcessed: 2,
					TotalBankTransactionsProcessed:   2,
					MatchedTransactions:              2,
				},
				DiscrepantTransactions: domain.DiscrepantTransactions{
					Count:                 2,
					TotalDiscrepancyValue: 0.55, // 0.05 + 0.50
					Details: []domain.DiscrepancyDetail{
						{
							SystemTransaction: domain.SystemTransaction{
								TrxID:           "TRX001",
								TransactionTime: baseTime.AddDate(0, 0, 1),
								Type:            domain.TransactionTypeDebit,
								Amount:          100.00,
							},
							BankTransaction: domain.BankTransaction{
								UniqueIdentifier: "BANK001",
								Date:             baseTime.AddDate(0, 0, 1),
								Type:             domain.TransactionTypeDebit,
								NormalizedAmount: 99.95,
								Description:      "Purchase trxID:TRX001",
								BankSource:       "Bank1",
							},
						},
						{
							SystemTransaction: domain.SystemTransaction{
								TrxID:           "TRX002",
								TransactionTime: baseTime.AddDate(0, 0, 2),
								Type:            domain.TransactionTypeCredit,
								Amount:          250.50,
							},
							BankTransaction: domain.BankTransaction{
								UniqueIdentifier: "BANK002",
								Date:             baseTime.AddDate(0, 0, 2),
								Type:             domain.TransactionTypeCredit,
								NormalizedAmount: 251.00,
								Description:      "Deposit trxID:TRX002",
								BankSource:       "Bank1",
							},
						},
					},
				},
				UnmatchedTransactions: domain.UnmatchedTransactions{
					BankMissingFromSystem: make(map[string][]domain.BankTransaction),
				},
			},
		},
		{
			name:       "reconciliation with unmatched transactions",
			systemPath: "/examples/transactions/system_transactions.csv",
			bankPaths:  []string{"/examples/statements/statement_bank_A.csv", "/examples/statements/statement_bank_B.csv"},
			start:      start,
			end:        end,
			systemTxs: []domain.SystemTransaction{
				{
					TrxID:           "TRX001",
					TransactionTime: baseTime.AddDate(0, 0, 1),
					Type:            domain.TransactionTypeDebit,
					Amount:          100.00,
				},
				{
					TrxID:           "TRX999", // No matching bank transaction
					TransactionTime: baseTime.AddDate(0, 0, 5),
					Type:            domain.TransactionTypeCredit,
					Amount:          500.00,
				},
			},
			bankTxs: []domain.BankTransaction{
				{
					UniqueIdentifier: "BANK001",
					Date:             baseTime.AddDate(0, 0, 1),
					Type:             domain.TransactionTypeDebit,
					NormalizedAmount: 100.00,
					Description:      "Purchase trxID:TRX001",
					BankSource:       "Bank1",
				},
				{
					UniqueIdentifier: "BANK999", // No matching system transaction
					Date:             baseTime.AddDate(0, 0, 4),
					Type:             domain.TransactionTypeDebit,
					NormalizedAmount: 75.00,
					Description:      "ATM Withdrawal",
					BankSource:       "Bank1",
				},
			},
			want: &domain.ReconciliationReport{
				ReconciliationSummary: domain.Summary{
					TimeframeStart:                   start.Format(time.DateOnly),
					TimeframeEnd:                     end.Format(time.DateOnly),
					TotalSystemTransactionsProcessed: 2,
					TotalBankTransactionsProcessed:   2,
					MatchedTransactions:              1,
				},
				DiscrepantTransactions: domain.DiscrepantTransactions{
					Details: make([]domain.DiscrepancyDetail, 0),
				},
				UnmatchedTransactions: domain.UnmatchedTransactions{
					Count: 2,
					SystemMissingFromBank: []domain.SystemTransaction{
						{
							TrxID:           "TRX999",
							TransactionTime: baseTime.AddDate(0, 0, 5),
							Type:            domain.TransactionTypeCredit,
							Amount:          500.00,
						},
					},
					BankMissingFromSystem: map[string][]domain.BankTransaction{
						"Bank1": {
							{
								UniqueIdentifier: "BANK999",
								Date:             baseTime.AddDate(0, 0, 4),
								Type:             domain.TransactionTypeDebit,
								NormalizedAmount: 75.00,
								Description:      "ATM Withdrawal",
								BankSource:       "Bank1",
							},
						},
					},
				},
			},
		},
		{
			name:       "group matching with multiple transactions",
			systemPath: "/examples/transactions/system_transactions.csv",
			bankPaths:  []string{"/examples/statements/statement_bank_A.csv", "/examples/statements/statement_bank_B.csv"},
			start:      start,
			end:        end,
			systemTxs: []domain.SystemTransaction{
				{
					TrxID:           "TRX001",
					TransactionTime: baseTime.AddDate(0, 0, 1),
					Type:            domain.TransactionTypeDebit,
					Amount:          100.00,
				},
				{
					TrxID:           "TRX002",
					TransactionTime: baseTime.AddDate(0, 0, 1),
					Type:            domain.TransactionTypeDebit,
					Amount:          100.00,
				},
			},
			bankTxs: []domain.BankTransaction{
				{
					UniqueIdentifier: "BANK001",
					Date:             baseTime.AddDate(0, 0, 1),
					Type:             domain.TransactionTypeDebit,
					NormalizedAmount: 100.00,
					Description:      "Purchase 1",
					BankSource:       "Bank1",
				},
				{
					UniqueIdentifier: "BANK002",
					Date:             baseTime.AddDate(0, 0, 1),
					Type:             domain.TransactionTypeDebit,
					NormalizedAmount: 100.00,
					Description:      "Purchase 2",
					BankSource:       "Bank1",
				},
			},
			want: &domain.ReconciliationReport{
				ReconciliationSummary: domain.Summary{
					TimeframeStart:                   start.Format(time.DateOnly),
					TimeframeEnd:                     end.Format(time.DateOnly),
					TotalSystemTransactionsProcessed: 2,
					TotalBankTransactionsProcessed:   2,
					MatchedTransactions:              2,
				},
				DiscrepantTransactions: domain.DiscrepantTransactions{
					Details: make([]domain.DiscrepancyDetail, 0),
				},
				UnmatchedTransactions: domain.UnmatchedTransactions{
					BankMissingFromSystem: make(map[string][]domain.BankTransaction),
				},
			},
		},
		{
			name:       "date filtering - transactions outside range excluded",
			systemPath: "/examples/transactions/system_transactions.csv",
			bankPaths:  []string{"/examples/statements/statement_bank_A.csv", "/examples/statements/statement_bank_B.csv"},
			start:      start,
			end:        end,
			systemTxs: []domain.SystemTransaction{
				{
					TrxID:           "TRX001",
					TransactionTime: baseTime.AddDate(0, 0, 1), // Within range
					Type:            domain.TransactionTypeDebit,
					Amount:          100.00,
				},
				{
					TrxID:           "TRX002",
					TransactionTime: baseTime.AddDate(0, 0, -1), // Before start
					Type:            domain.TransactionTypeCredit,
					Amount:          250.50,
				},
				{
					TrxID:           "TRX003",
					TransactionTime: baseTime.AddDate(0, 0, 10), // After end
					Type:            domain.TransactionTypeDebit,
					Amount:          75.25,
				},
			},
			bankTxs: []domain.BankTransaction{
				{
					UniqueIdentifier: "BANK001",
					Date:             baseTime.AddDate(0, 0, 1), // Within range
					Type:             domain.TransactionTypeDebit,
					NormalizedAmount: 100.00,
					Description:      "Purchase trxID:TRX001",
					BankSource:       "Bank1",
				},
				{
					UniqueIdentifier: "BANK002",
					Date:             baseTime.AddDate(0, 0, -1), // Before start
					Type:             domain.TransactionTypeCredit,
					NormalizedAmount: 250.50,
					Description:      "Deposit",
					BankSource:       "Bank1",
				},
			},
			want: &domain.ReconciliationReport{
				ReconciliationSummary: domain.Summary{
					TimeframeStart:                   start.Format(time.DateOnly),
					TimeframeEnd:                     end.Format(time.DateOnly),
					TotalSystemTransactionsProcessed: 1,
					TotalBankTransactionsProcessed:   1,
					MatchedTransactions:              1,
				},
				DiscrepantTransactions: domain.DiscrepantTransactions{
					Details: make([]domain.DiscrepancyDetail, 0),
				},
				UnmatchedTransactions: domain.UnmatchedTransactions{
					BankMissingFromSystem: make(map[string][]domain.BankTransaction),
				},
			},
		},
		{
			name:            "system repository error",
			systemPath:      "/examples/transactions/system_transactions.csv",
			bankPaths:       []string{"/examples/statements/statement_bank_A.csv", "/examples/statements/statement_bank_B.csv"},
			start:           start,
			end:             end,
			systemRepoError: errors.New("failed to read system transactions"),
			wantErr:         true,
		},
		{
			name:          "bank repository error",
			systemPath:    "/examples/transactions/system_transactions.csv",
			bankPaths:     []string{"/examples/statements/statement_bank_A.csv", "/examples/statements/statement_bank_B.csv"},
			start:         start,
			end:           end,
			systemTxs:     []domain.SystemTransaction{},
			bankRepoError: errors.New("failed to read bank transactions"),
			wantErr:       true,
		},
		{
			name:       "empty transactions",
			systemPath: "/examples/transactions/system_transactions.csv",
			bankPaths:  []string{"/examples/statements/statement_bank_A.csv", "/examples/statements/statement_bank_B.csv"},
			start:      start,
			end:        end,
			systemTxs:  []domain.SystemTransaction{},
			bankTxs:    []domain.BankTransaction{},
			want: &domain.ReconciliationReport{
				ReconciliationSummary: domain.Summary{
					TimeframeStart:                   start.Format(time.DateOnly),
					TimeframeEnd:                     end.Format(time.DateOnly),
					TotalSystemTransactionsProcessed: 0,
					TotalBankTransactionsProcessed:   0,
					MatchedTransactions:              0,
				},
				DiscrepantTransactions: domain.DiscrepantTransactions{
					Details: make([]domain.DiscrepancyDetail, 0),
				},
				UnmatchedTransactions: domain.UnmatchedTransactions{
					BankMissingFromSystem: make(map[string][]domain.BankTransaction),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mTransactionRepo := mock_usecase.NewMockTransactionRepository(ctrl)

			// Setup mock expectations
			if tt.systemRepoError != nil {
				mTransactionRepo.EXPECT().
					GetSystemTransactions(gomock.Any(), tt.systemPath).
					Return(nil, tt.systemRepoError)
			} else {
				mTransactionRepo.EXPECT().
					GetSystemTransactions(gomock.Any(), tt.systemPath).
					Return(tt.systemTxs, nil)

				if tt.bankRepoError != nil {
					mTransactionRepo.EXPECT().
						GetBankTransactions(gomock.Any(), tt.bankPaths).
						Return(nil, tt.bankRepoError)
				} else {
					mTransactionRepo.EXPECT().
						GetBankTransactions(gomock.Any(), tt.bankPaths).
						Return(tt.bankTxs, nil)
				}
			}

			uc := usecase.NewReconciliationUseCase(mTransactionRepo)
			got, gotErr := uc.Reconcile(context.Background(), tt.systemPath, tt.bankPaths, tt.start, tt.end)

			if tt.wantErr {
				assert.Error(t, gotErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, gotErr)
				assert.NotNil(t, got)

				// Compare the reports carefully
				assert.Equal(t, tt.want.ReconciliationSummary, got.ReconciliationSummary)
				assert.Equal(t, tt.want.DiscrepantTransactions.Count, got.DiscrepantTransactions.Count)
				assert.InDelta(t, tt.want.DiscrepantTransactions.TotalDiscrepancyValue, got.DiscrepantTransactions.TotalDiscrepancyValue, 0.001)
				assert.Equal(t, len(tt.want.DiscrepantTransactions.Details), len(got.DiscrepantTransactions.Details))

				assert.Equal(t, tt.want.UnmatchedTransactions.Count, got.UnmatchedTransactions.Count)
				assert.Equal(t, tt.want.UnmatchedTransactions.SystemMissingFromBank, got.UnmatchedTransactions.SystemMissingFromBank)
				assert.Equal(t, len(tt.want.UnmatchedTransactions.BankMissingFromSystem), len(got.UnmatchedTransactions.BankMissingFromSystem))
			}

		})
	}
}
