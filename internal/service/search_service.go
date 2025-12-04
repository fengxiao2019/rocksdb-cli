package service

import (
	"rocksdb-cli/internal/db"
)

// SearchService provides advanced search operations
type SearchService struct {
	db db.KeyValueDB
}

// SearchOptions contains options for search operations
type SearchOptions struct {
	KeyPattern    string `json:"key_pattern"`    // Pattern to search in keys
	ValuePattern  string `json:"value_pattern"`  // Pattern to search in values
	StartKey      string `json:"start_key"`      // Start of key range (inclusive)
	EndKey        string `json:"end_key"`        // End of key range (exclusive)
	UseRegex      bool   `json:"use_regex"`      // Whether to use regex matching
	CaseSensitive bool   `json:"case_sensitive"` // Whether search is case sensitive
	KeysOnly      bool   `json:"keys_only"`      // Return only keys, not values
	Limit         int    `json:"limit"`          // Maximum number of results
	After         string `json:"after"`          // Cursor for pagination
}

// SearchResult contains the results of a search operation
type SearchResult struct {
	Results    []SearchResultItem `json:"results"`     // Search results
	Count      int                `json:"count"`       // Number of results in this page
	Total      int                `json:"total"`       // Total number of results found
	HasMore    bool               `json:"has_more"`    // Whether more results exist
	NextCursor string             `json:"next_cursor"` // Last key in this page
	QueryTime  string             `json:"query_time"`  // Time taken for the search
}

// SearchResultItem represents a single search result
type SearchResultItem struct {
	Key           string   `json:"key"`
	Value         string   `json:"value"`
	KeyIsBinary   bool     `json:"key_is_binary"`   // true if key is hex encoded
	ValueIsBinary bool     `json:"value_is_binary"` // true if value is hex encoded
	Timestamp     string   `json:"timestamp"`       // parsed timestamp if key is a timestamp
	MatchedFields []string `json:"matched_fields"`  // Which fields matched (key, value, both)
}

// JSONQueryResult contains the results of a JSON field query
type JSONQueryResult struct {
	Data  map[string]string `json:"data"`  // Key-value pairs
	Count int               `json:"count"` // Number of results
	Field string            `json:"field"` // Field that was queried
	Value string            `json:"value"` // Value that was searched for
}

// NewSearchService creates a new SearchService instance
func NewSearchService(database db.KeyValueDB) *SearchService {
	return &SearchService{db: database}
}

// Search performs an advanced search on a column family
func (s *SearchService) Search(cf string, opts SearchOptions) (*SearchResult, error) {
	// Convert service options to db options
	dbOpts := db.SearchOptions{
		KeyPattern:    opts.KeyPattern,
		ValuePattern:  opts.ValuePattern,
		StartKey:      opts.StartKey,
		EndKey:        opts.EndKey,
		UseRegex:      opts.UseRegex,
		CaseSensitive: opts.CaseSensitive,
		KeysOnly:      opts.KeysOnly,
		Limit:         opts.Limit,
		After:         opts.After,
	}

	results, err := s.db.SearchCF(cf, dbOpts)
	if err != nil {
		return nil, err
	}

	// Convert db results to service results
	items := make([]SearchResultItem, 0, len(results.Results))
	for _, r := range results.Results {
		items = append(items, SearchResultItem{
			Key:           r.Key,
			Value:         r.Value,
			KeyIsBinary:   r.KeyIsBinary,
			ValueIsBinary: r.ValueIsBinary,
			Timestamp:     r.Timestamp,
			MatchedFields: r.MatchedFields,
		})
	}

	return &SearchResult{
		Results:    items,
		Count:      len(items),
		Total:      results.Total,
		HasMore:    results.HasMore,
		NextCursor: results.NextCursor,
		QueryTime:  results.QueryTime,
	}, nil
}

// JSONQuery performs a query on JSON field values
func (s *SearchService) JSONQuery(cf, field, value string) (*JSONQueryResult, error) {
	data, err := s.db.JSONQueryCF(cf, field, value)
	if err != nil {
		return nil, err
	}

	return &JSONQueryResult{
		Data:  data,
		Count: len(data),
		Field: field,
		Value: value,
	}, nil
}
