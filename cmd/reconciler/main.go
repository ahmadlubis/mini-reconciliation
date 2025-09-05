package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"mini-reconciliation/internal/gateway"
	"mini-reconciliation/internal/usecase"
)

func main() {
	// Define command-line flags
	systemFile := flag.String("system", "", "Path to the system transactions CSV file (required)")
	bankFilesStr := flag.String("bank", "", "Comma-separated list of paths to bank statement CSV files (required)")
	startDateStr := flag.String("start", "", "Start date for reconciliation (YYYY-MM-DD) (required)")
	endDateStr := flag.String("end", "", "End date for reconciliation (YYYY-MM-DD) (required)")
	flag.Parse()

	// Validate required flags
	if *systemFile == "" || *bankFilesStr == "" || *startDateStr == "" || *endDateStr == "" {
		fmt.Println("Error: All flags (-system, -bank, -start, -end) are required.")
		flag.Usage()
		os.Exit(1)
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", *startDateStr)
	if err != nil {
		log.Fatalf("Error parsing start date: %v", err)
	}
	endDate, err := time.Parse("2006-01-02", *endDateStr)
	if err != nil {
		log.Fatalf("Error parsing end date: %v", err)
	}

	// Split bank files string into a slice
	bankFiles := strings.Split(*bankFilesStr, ",")

	// --- Dependency Injection (Wiring the application) ---
	// In a larger app, this might be done with a DI container.
	// Here, we do it manually, which is clear and simple.

	// 1. Create the repository (the outermost layer)
	csvRepo := gateway.NewCSVTransactionRepository()

	// 2. Create the usecase and inject the repository (the core logic layer)
	reconciliationUseCase := usecase.NewReconciliationUseCase(csvRepo)

	// --- Execute the Usecase ---
	report, err := reconciliationUseCase.Reconcile(context.Background(), *systemFile, bankFiles, startDate, endDate)
	if err != nil {
		log.Fatalf("Reconciliation failed: %v", err)
	}

	// --- Present the Output ---
	output, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatalf("Failed to generate JSON report: %v", err)
	}

	fmt.Println(string(output))
}
