# mini-reconciliation

A small command-line Go application that reconciles your internal (system) transactions against one or more external bank statement CSV files and prints a reconciliation summary in JSON.

## Table of Contents

- [What it does](#what-it-does)
- [Features](#features)
- [Project Layout](#project-layout)
- [Prerequisites](#prerequisites)
- [Build / Install](#build--install)
- [Usage](#usage)
- [CSV Formats (Expected)](#csv-formats-expected)
- [Output (JSON) — Example Shape](#output-json--example-shape)
- [Examples](#examples)
- [Development Notes & Tests](#development-notes--tests)
- [Contributing](#contributing)
- [License](#license)

## What it does

`mini-reconciliation` compares a system (internal) transaction CSV against one or more bank statement CSVs and reports:

- **Matched transactions** (system ↔ bank)
- **Transactions present only in the system**
- **Transactions present only in the bank statement(s)**
- **Possible discrepancies** (e.g. amounts or dates differ)

## Features

- Single binary CLI (no server required)
- Accepts multiple bank statement CSV files
- Date range filtering (start / end)
- Outputs a JSON reconciliation report suitable for further processing/automation

## Project Layout

```
/
├─ cmd/reconciler        # CLI entrypoint
├─ internal/
│  ├─ domain             # core business entities (transactions, report models)
│  ├─ usecase            # reconciliation logic
│  └─ gateway            # CSV readers / adapters
├─ examples              # sample CSV files (see examples/...)
├─ README.md
├─ go.mod
└─ LICENSE
```

This follows a Clean Architecture style (separation of domain, usecases, gateways).

## Prerequisites

- Go 1.21 or newer (project references Go modules)

## Build / Install

From the repository root:

**Install & Vendor Dependencies:**
```bash
make dep
```

**Build a local binary:**
```bash
make build
```

**Or install to your `$GOBIN` (if you want reconciler on your PATH):**
```bash
make install
```

## Usage

The CLI expects:

- `-system` — path to the system (internal) transactions CSV
- `-bank` — comma-separated list of bank statement CSV file paths
- `-start` — start date (YYYY-MM-DD)
- `-end` — end date (YYYY-MM-DD)

**Example:**
```bash
./reconciler \
  -system="examples/transactions/system_transactions.csv" \
  -bank="examples/statements/statement_bank_A.csv,examples/statements/statement_bank_B.csv" \
  -start="2025-09-01" \
  -end="2025-09-05"
```

**Notes:**
- `-bank` accepts multiple comma-separated file paths (so you can reconcile a single system file against many bank statements)
- Date filtering uses the YYYY-MM-DD format

## CSV Formats (Expected)

There are many ways to format CSVs. The CLI reads CSVs from `examples/` in the repo. If you adapt your own CSVs, make sure they contain at least:

### System (internal) CSV — minimal required columns
```csv
id,date,amount,description
```

### Bank statement CSV — minimal required columns
```csv
id,date,amount,description
```

**Requirements:**
- `date` should be in a parseable ISO-like format (e.g. YYYY-MM-DD or YYYY-MM-DDTHH:MM:SS)
- `amount` should be a numeric value; debit/credit conventions vary—ensure your file consistently uses positive/negative or single-sided format
- If your bank CSV includes extra columns (bank reference number, balance, currency), the gateway/adapter code should ignore or map them—check the example CSVs in `examples/` to confirm exact headings and order

## Output (JSON) — Example Shape

The CLI prints a JSON reconciliation report to STDOUT. A typical structure looks like:

```json
{
  "summary": {
    "period": { "start": "2025-09-01", "end": "2025-09-05" },
    "system_count": 123,
    "bank_count": 200,
    "matched_count": 110,
    "system_only_count": 13,
    "bank_only_count": 90
  },
  "matched": [
    { 
      "system_id": "S123", 
      "bank_id": "B987", 
      "date": "2025-09-02", 
      "amount": 100.00 
    }
  ],
  "system_only": [
    { 
      "system_id": "S124", 
      "date": "2025-09-03", 
      "amount": 50.00, 
      "description": "Refund" 
    }
  ],
  "bank_only": [
    { 
      "bank_id": "B988", 
      "date": "2025-09-04", 
      "amount": 75.00, 
      "description": "Fee" 
    }
  ],
  "discrepancies": [
    { 
      "system_id": "S125", 
      "bank_id": "B989", 
      "system_amount": 100.00, 
      "bank_amount": 99.99, 
      "note": "amount mismatch" 
    }
  ]
}
```

> **Note:** The exact keys and structure may differ slightly; run the binary with your files to see the concrete output and to confirm keys.

## Examples

Run the example from the repo:

```bash
go build -o reconciler ./cmd/reconciler

./reconciler \
  -system="examples/transactions/system_transactions.csv" \
  -bank="examples/statements/statement_bank_A.csv,examples/statements/statement_bank_B.csv" \
  -start="2025-09-01" \
  -end="2025-09-05"
```

The command prints JSON to the console.