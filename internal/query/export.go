package query

import (
	"encoding/csv"
	"encoding/json"
	"os"

	"github.com/tardanoir/seshat/internal/db"
)

func ExportCSV(result *db.QueryResult, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write(result.Columns); err != nil {
		return err
	}
	for _, row := range result.Rows {
		clean := make([]string, len(row))
		for i, v := range row {
			if v == "NULL" {
				clean[i] = ""
			} else {
				clean[i] = v
			}
		}
		if err := w.Write(clean); err != nil {
			return err
		}
	}
	return w.Error()
}

func ExportJSON(result *db.QueryResult, path string) error {
	records := make([]map[string]any, len(result.Rows))
	for i, row := range result.Rows {
		record := make(map[string]any)
		for j, col := range result.Columns {
			if j < len(row) {
				if row[j] == "NULL" {
					record[col] = nil
				} else {
					record[col] = row[j]
				}
			}
		}
		records[i] = record
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
