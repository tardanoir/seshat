package query

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/tardanoir/seshat/internal/config"
)

// add search in the history

const maxHistoryEntries = 100

type HistoryEntry struct {
	SQL        string    `json:"sql"`
	Connection string    `json:"connection"`
	Timestamp  time.Time `json:"timestamp"`
	Duration   string    `json:"duration"`
	RowCount   int       `json:"row_count"`
}

func historyPath() string {
	return filepath.Join(config.ConfigDir(), "history.json")
}

func LoadHistory() ([]HistoryEntry, error) {
	data, err := os.ReadFile(historyPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []HistoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func AddHistory(entry HistoryEntry) error {
	entries, _ := LoadHistory()
	entries = append([]HistoryEntry{entry}, entries...)
	if len(entries) > maxHistoryEntries {
		entries = entries[:maxHistoryEntries]
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(historyPath(), data, 0o600)
}
