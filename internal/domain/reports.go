package domain

// DiscrepancyDetail provides details on a single discrepant transaction.
type DiscrepancyDetail struct {
	SystemTransaction SystemTransaction `json:"system_transaction"`
	BankTransaction   BankTransaction   `json:"bank_transaction"`
}

// DiscrepantTransactions holds summary information about all discrepancies found.
type DiscrepantTransactions struct {
	Count                 int                 `json:"count"`
	TotalDiscrepancyValue float64             `json:"total_discrepancy_value"`
	Details               []DiscrepancyDetail `json:"details"`
}

// UnmatchedTransactions lists all transactions that could not be matched.
type UnmatchedTransactions struct {
	Count                 int                          `json:"count"`
	SystemMissingFromBank []SystemTransaction          `json:"system_missing_from_bank"`
	BankMissingFromSystem map[string][]BankTransaction `json:"bank_missing_from_system"`
}

// Summary provides high-level statistics of the reconciliation process.
type Summary struct {
	TimeframeStart                   string `json:"timeframe_start"`
	TimeframeEnd                     string `json:"timeframe_end"`
	TotalSystemTransactionsProcessed int    `json:"total_system_transactions_processed"`
	TotalBankTransactionsProcessed   int    `json:"total_bank_transactions_processed"`
	MatchedTransactions              int    `json:"matched_transactions"`
}

// ReconciliationReport is the top-level structure for the final JSON output.
type ReconciliationReport struct {
	ReconciliationSummary  Summary                `json:"reconciliation_summary"`
	DiscrepantTransactions DiscrepantTransactions `json:"discrepant_transactions"`
	UnmatchedTransactions  UnmatchedTransactions  `json:"unmatched_transactions"`
}
