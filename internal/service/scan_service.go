package service

import (
	"rocksdb-cli/internal/db"
)

// KeyValue represents a single key-value pair with binary encoding info
type KeyValue = db.KeyValue

// ScanService provides scanning and querying operations
type ScanService struct {
	db db.KeyValueDB
}

// ScanOptions contains options for scanning operations
type ScanOptions struct {
	StartKey  string `json:"start_key"`  // Start key for range scan
	EndKey    string `json:"end_key"`    // End key for range scan (exclusive)
	Limit     int    `json:"limit"`      // Maximum number of results
	Reverse   bool   `json:"reverse"`    // Scan in reverse order
	KeysOnly  bool   `json:"keys_only"`  // Return only keys without values
	After     string `json:"after"`      // Cursor for pagination (start after this key)
}

// ScanResult contains the results of a scan operation
type ScanResult struct {
	Data       map[string]string `json:"data"`        // Key-value pairs (deprecated, use DataV2)
	ResultsV2  []KeyValue        `json:"results_v2"`  // New format with binary support
	Count      int               `json:"count"`       // Number of results in this page
	HasMore    bool              `json:"has_more"`    // Whether more results exist
	NextCursor string            `json:"next_cursor"` // Last key in this page, use as "after" for next page
}

// PrefixScanResult contains the results of a prefix scan operation
type PrefixScanResult struct {
	Data  map[string]string `json:"data"`  // Key-value pairs
	Count int               `json:"count"` // Number of results
}

// NewScanService creates a new ScanService instance
func NewScanService(database db.KeyValueDB) *ScanService {
	return &ScanService{db: database}
}

// Scan performs a range scan on a column family
func (s *ScanService) Scan(cf string, opts ScanOptions) (*ScanResult, error) {
	// Convert service options to db options
	dbOpts := db.ScanOptions{
		Limit:      opts.Limit,
		Reverse:    opts.Reverse,
		Values:     !opts.KeysOnly,
		StartAfter: opts.After,
	}

	// Use SmartScanCFPage for paginated results
	pageResult, err := s.db.SmartScanCFPage(cf, opts.StartKey, opts.EndKey, dbOpts)
	if err != nil {
		return nil, err
	}

	return &ScanResult{
		Data:       pageResult.Results,
		ResultsV2:  pageResult.ResultsV2,
		Count:      len(pageResult.Results),
		HasMore:    pageResult.HasMore,
		NextCursor: pageResult.NextCursor,
	}, nil
}

// PrefixScan performs a prefix scan on a column family
func (s *ScanService) PrefixScan(cf, prefix string, limit int) (*PrefixScanResult, error) {
	data, err := s.db.SmartPrefixScanCF(cf, prefix, limit)
	if err != nil {
		return nil, err
	}

	return &PrefixScanResult{
		Data:  data,
		Count: len(data),
	}, nil
}

// GetAll retrieves all key-value pairs from a column family
// Warning: Use with caution on large datasets
func (s *ScanService) GetAll(cf string, limit int) (*ScanResult, error) {
	return s.Scan(cf, ScanOptions{
		StartKey: "",
		EndKey:   "",
		Limit:    limit,
		KeysOnly: false,
	})
}
