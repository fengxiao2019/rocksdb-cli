package service

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"rocksdb-cli/internal/db"
)

// ExportService provides data export operations
type ExportService struct {
	db db.KeyValueDB
}

// ExportOptions contains options for export operations
type ExportOptions struct {
	CF        string   `json:"cf"`        // Column family to export
	Format    string   `json:"format"`    // Export format: "csv" or "json"
	Keys      []string `json:"keys"`      // Optional: specific keys to export (if empty, export all)
	Separator string   `json:"separator"` // CSV separator (default: ",")
	Header    bool     `json:"header"`    // Include header row in CSV
	Pretty    bool     `json:"pretty"`    // Pretty-print JSON
}

// ExportResult contains the result of an export operation
type ExportResult struct {
	RecordCount int    `json:"record_count"` // Number of records exported
	BytesWritten int64 `json:"bytes_written"` // Number of bytes written
}

// NewExportService creates a new ExportService instance
func NewExportService(database db.KeyValueDB) *ExportService {
	return &ExportService{db: database}
}

// ExportToCSV exports data to CSV format
func (s *ExportService) ExportToCSV(w io.Writer, opts ExportOptions) (*ExportResult, error) {
	writer := csv.NewWriter(w)
	if opts.Separator != "" && len(opts.Separator) > 0 {
		writer.Comma = rune(opts.Separator[0])
	}
	defer writer.Flush()

	recordCount := 0

	// Write header if requested
	if opts.Header {
		if err := writer.Write([]string{"Key", "Value"}); err != nil {
			return nil, err
		}
	}

	// Get data
	data, err := s.getData(opts.CF, opts.Keys)
	if err != nil {
		return nil, err
	}

	// Write data
	for k, v := range data {
		if err := writer.Write([]string{k, v}); err != nil {
			return nil, err
		}
		recordCount++
	}

	return &ExportResult{
		RecordCount: recordCount,
	}, nil
}

// ExportToJSON exports data to JSON format
func (s *ExportService) ExportToJSON(w io.Writer, opts ExportOptions) (*ExportResult, error) {
	data, err := s.getData(opts.CF, opts.Keys)
	if err != nil {
		return nil, err
	}

	encoder := json.NewEncoder(w)
	if opts.Pretty {
		encoder.SetIndent("", "  ")
	}

	if err := encoder.Encode(data); err != nil {
		return nil, err
	}

	return &ExportResult{
		RecordCount: len(data),
	}, nil
}

// ExportSearchResults exports search results to CSV
func (s *ExportService) ExportSearchResults(w io.Writer, cf string, searchOpts SearchOptions, csvSep string) (*ExportResult, error) {
	// Convert service search options to db search options
	dbOpts := db.SearchOptions{
		KeyPattern:    searchOpts.KeyPattern,
		ValuePattern:  searchOpts.ValuePattern,
		UseRegex:      searchOpts.UseRegex,
		CaseSensitive: searchOpts.CaseSensitive,
		KeysOnly:      searchOpts.KeysOnly,
		Limit:         searchOpts.Limit,
		After:         searchOpts.After,
	}

	// Use the database's built-in export method
	if err := s.db.ExportSearchResultsToCSV(cf, "", csvSep, dbOpts); err != nil {
		return nil, err
	}

	// Note: The current implementation doesn't return the count
	// This could be improved by getting the search results first
	return &ExportResult{
		RecordCount: 0, // Unknown without additional query
	}, nil
}

// getData retrieves data from the database based on the provided keys or all data
func (s *ExportService) getData(cf string, keys []string) (map[string]string, error) {
	if len(keys) > 0 {
		// Export specific keys
		result := make(map[string]string)
		for _, key := range keys {
			val, err := s.db.SmartGetCF(cf, key)
			if err != nil {
				// Skip keys that don't exist
				if err == db.ErrKeyNotFound {
					continue
				}
				return nil, err
			}
			result[key] = val
		}
		return result, nil
	}

	// Export all data
	return s.db.SmartScanCF(cf, "", "", db.ScanOptions{Values: true})
}
