package usecase

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"mini-reconciliation/internal/domain"
)

// ReconciliationUseCase orchestrates the reconciliation process.
type ReconciliationUseCase struct {
	repo TransactionRepository
}

// NewReconciliationUseCase creates a new instance of the usecase.
func NewReconciliationUseCase(repo TransactionRepository) *ReconciliationUseCase {
	return &ReconciliationUseCase{repo: repo}
}

// Reconcile performs the main reconciliation logic.
func (uc *ReconciliationUseCase) Reconcile(ctx context.Context, systemPath string, bankPaths []string, start, end time.Time) (*domain.ReconciliationReport, error) {
	// Step 1: Data Ingestion
	systemTransactions, err := uc.repo.GetSystemTransactions(ctx, systemPath)
	if err != nil {
		return nil, fmt.Errorf("could not get system transactions: %w", err)
	}

	bankTransactions, err := uc.repo.GetBankTransactions(ctx, bankPaths)
	if err != nil {
		return nil, fmt.Errorf("could not get bank transactions: %w", err)
	}

	// Step 2: Timeframe Filtering
	filteredSystemTx := filterSystemTransactionsByDate(systemTransactions, start, end)
	filteredBankTx := filterBankTransactionsByDate(bankTransactions, start, end)

	report := domain.ReconciliationReport{
		ReconciliationSummary: domain.Summary{
			TimeframeStart:                   start.Format(time.DateOnly),
			TimeframeEnd:                     end.Format(time.DateOnly),
			TotalSystemTransactionsProcessed: len(filteredSystemTx),
			TotalBankTransactionsProcessed:   len(filteredBankTx),
		},
		DiscrepantTransactions: domain.DiscrepantTransactions{
			Details: make([]domain.DiscrepancyDetail, 0),
		},
		UnmatchedTransactions: domain.UnmatchedTransactions{
			BankMissingFromSystem: make(map[string][]domain.BankTransaction),
		},
	}

	// Step 3: Multi-Pass Matching Strategy
	matchedSystem := make(map[string]bool)
	matchedBank := make(map[string]bool)

	// Pass 1: Unique Identifier Matching (trxID in description)
	for _, bankTx := range filteredBankTx {
		for _, sysTx := range filteredSystemTx {
			if matchedSystem[sysTx.TrxID] || matchedBank[bankTx.UniqueIdentifier] {
				continue
			}
			if strings.Contains(bankTx.Description, "trxID:"+sysTx.TrxID) {
				uc.processMatch(&report, sysTx, bankTx)
				matchedSystem[sysTx.TrxID] = true
				matchedBank[bankTx.UniqueIdentifier] = true
			}
		}
	}

	// Pass 2 & 3: Exact and Group Matching
	// Create a map to group transactions by a composite key of date, type, and amount.
	type groupKey string
	systemMap := make(map[groupKey][]domain.SystemTransaction)
	bankMap := make(map[groupKey][]domain.BankTransaction)

	for _, sysTx := range filteredSystemTx {
		if !matchedSystem[sysTx.TrxID] {
			key := groupKey(buildGroupKey(sysTx.TransactionTime, sysTx.Type, sysTx.Amount))
			systemMap[key] = append(systemMap[key], sysTx)
		}
	}
	for _, bankTx := range filteredBankTx {
		if !matchedBank[bankTx.UniqueIdentifier] {
			key := groupKey(buildGroupKey(bankTx.Date, bankTx.Type, bankTx.NormalizedAmount))
			bankMap[key] = append(bankMap[key], bankTx)
		}
	}

	for key, sysTxs := range systemMap {
		bankTxs, ok := bankMap[key]
		if ok && len(sysTxs) == len(bankTxs) { // Pass 2 (len=1) and Pass 3 (len>1)
			for i := 0; i < len(sysTxs); i++ {
				uc.processMatch(&report, sysTxs[i], bankTxs[i])
				matchedSystem[sysTxs[i].TrxID] = true
				matchedBank[bankTxs[i].UniqueIdentifier] = true
			}
			// Remove matched items from maps
			delete(systemMap, key)
			delete(bankMap, key)
		}
	}

	// Step 4: Collate Unmatched Transactions
	for _, sysTxs := range systemMap {
		report.UnmatchedTransactions.SystemMissingFromBank = append(report.UnmatchedTransactions.SystemMissingFromBank, sysTxs...)
	}
	for _, bankTxs := range bankMap {
		for _, bankTx := range bankTxs {
			report.UnmatchedTransactions.BankMissingFromSystem[bankTx.BankSource] = append(report.UnmatchedTransactions.BankMissingFromSystem[bankTx.BankSource], bankTx)
		}
	}

	// FIXED: Calculate count AFTER populating unmatched transactions
	report.UnmatchedTransactions.Count = len(report.UnmatchedTransactions.SystemMissingFromBank) + countBankMapItems(report.UnmatchedTransactions.BankMissingFromSystem)

	return &report, nil
}

// processMatch handles a matched pair, checking for discrepancies.
func (uc *ReconciliationUseCase) processMatch(report *domain.ReconciliationReport, sysTx domain.SystemTransaction, bankTx domain.BankTransaction) {
	report.ReconciliationSummary.MatchedTransactions++
	// Check for discrepancy (using a small epsilon for float comparison)
	if math.Abs(sysTx.Amount-bankTx.NormalizedAmount) > 0.001 {
		diff := math.Abs(sysTx.Amount - bankTx.NormalizedAmount)
		report.DiscrepantTransactions.Count++
		report.DiscrepantTransactions.TotalDiscrepancyValue += diff
		report.DiscrepantTransactions.Details = append(report.DiscrepantTransactions.Details, domain.DiscrepancyDetail{
			SystemTransaction: sysTx,
			BankTransaction:   bankTx,
		})
	}
}

func buildGroupKey(t time.Time, txType domain.TransactionType, amount float64) string {
	return string(fmt.Sprintf("%s-%s-%.2f", t.Format("2006-01-02"), txType, amount))
}

func filterSystemTransactionsByDate(transactions []domain.SystemTransaction, start, end time.Time) []domain.SystemTransaction {
	var filtered []domain.SystemTransaction
	for _, tx := range transactions {
		txDate := tx.TransactionTime
		if (txDate.Equal(start) || txDate.After(start)) && (txDate.Equal(end) || txDate.Before(end.Add(24*time.Hour-time.Hour))) {
			filtered = append(filtered, tx)
		}
	}
	return filtered
}

func filterBankTransactionsByDate(transactions []domain.BankTransaction, start, end time.Time) []domain.BankTransaction {
	var filtered []domain.BankTransaction
	for _, tx := range transactions {
		txDate := tx.Date
		// Match the same logic as system transactions for consistency
		if (txDate.Equal(start) || txDate.After(start)) && (txDate.Equal(end) || txDate.Before(end.Add(24*time.Hour-time.Nanosecond))) {
			filtered = append(filtered, tx)
		}
	}
	return filtered
}

func countBankMapItems(m map[string][]domain.BankTransaction) int {
	count := 0
	for _, v := range m {
		count += len(v)
	}
	return count
}
