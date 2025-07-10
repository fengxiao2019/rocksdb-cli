package db

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"rocksdb-cli/internal/util"

	"encoding/binary"

	"github.com/linxGnu/grocksdb"
)

// Specific error types for better error handling
var (
	ErrKeyNotFound          = errors.New("key not found")
	ErrColumnFamilyNotFound = errors.New("column family not found")
	ErrColumnFamilyExists   = errors.New("column family already exists")
	ErrReadOnlyMode         = errors.New("operation not allowed in read-only mode")
	ErrColumnFamilyEmpty    = errors.New("column family is empty")
	ErrDatabaseClosed       = errors.New("database is closed")
)

// DataType represents the detected data type of a value
type DataType string

const (
	DataTypeJSON      DataType = "JSON"
	DataTypeNumber    DataType = "Number"
	DataTypeTimestamp DataType = "Timestamp"
	DataTypeString    DataType = "String"
	DataTypeBinary    DataType = "Binary"
	DataTypeEmpty     DataType = "Empty"
)

// CFStats contains statistics for a column family
type CFStats struct {
	Name                    string             `json:"name"`
	KeyCount                int64              `json:"key_count"`
	TotalKeySize            int64              `json:"total_key_size"`
	TotalValueSize          int64              `json:"total_value_size"`
	AverageKeySize          float64            `json:"average_key_size"`
	AverageValueSize        float64            `json:"average_value_size"`
	DataTypeDistribution    map[DataType]int64 `json:"data_type_distribution"`
	KeyLengthDistribution   map[string]int64   `json:"key_length_distribution"`
	ValueLengthDistribution map[string]int64   `json:"value_length_distribution"`
	CommonPrefixes          map[string]int64   `json:"common_prefixes"`
	SampleKeys              []string           `json:"sample_keys"`
	LastUpdated             time.Time          `json:"last_updated"`
}

// DatabaseStats contains overall database statistics
type DatabaseStats struct {
	ColumnFamilies    []CFStats `json:"column_families"`
	TotalKeyCount     int64     `json:"total_key_count"`
	TotalSize         int64     `json:"total_size"`
	ColumnFamilyCount int       `json:"column_family_count"`
	LastUpdated       time.Time `json:"last_updated"`
}

// SearchOptions contains options for fuzzy search operations
type SearchOptions struct {
	KeyPattern    string `json:"key_pattern"`    // Pattern to search in keys
	ValuePattern  string `json:"value_pattern"`  // Pattern to search in values
	UseRegex      bool   `json:"use_regex"`      // Whether to use regex matching
	CaseSensitive bool   `json:"case_sensitive"` // Whether search is case sensitive
	Limit         int    `json:"limit"`          // Maximum number of results
	KeysOnly      bool   `json:"keys_only"`      // Return only keys, not values
	After         string `json:"after"`          // Cursor for pagination
	Tick          bool   `json:"tick"`           // Whether to treat keys as .NET tick times and convert to UTC string
}

// SearchResult contains a single search result
type SearchResult struct {
	Key           string   `json:"key"`
	Value         string   `json:"value"`
	MatchedFields []string `json:"matched_fields"` // Which fields matched (key, value, both)
}

// SearchResults contains search results and metadata
type SearchResults struct {
	Results    []SearchResult `json:"results"`
	Total      int            `json:"total"`
	Limited    bool           `json:"limited"`     // Whether results were limited
	QueryTime  string         `json:"query_time"`  // Time taken for the search
	NextCursor string         `json:"next_cursor"` // Last key in this page, or "" if no more
	HasMore    bool           `json:"has_more"`    // True if more results exist
}

// ScanPageResult contains paginated scan results
// NextCursor is the last key in this page, or "" if no more
// HasMore is true if more results exist
type ScanPageResult struct {
	Results    map[string]string
	NextCursor string
	HasMore    bool
}

// ScanOptions now supports cursor-based pagination
// StartAfter: skip all keys <= this key (for forward scan)
type ScanOptions struct {
	Limit      int
	Reverse    bool
	Values     bool
	StartAfter string // cursor for pagination
}

type KeyValueDB interface {
	GetCF(cf, key string) (string, error)
	PutCF(cf, key, value string) error
	PrefixScanCF(cf, prefix string, limit int) (map[string]string, error)
	ScanCF(cf string, start, end []byte, opts ScanOptions) (map[string]string, error)
	ScanCFPage(cf string, start, end []byte, opts ScanOptions) (ScanPageResult, error) // new paginated version
	GetLastCF(cf string) (string, string, error)                                       // Returns key, value, error
	ExportToCSV(cf, filePath, sep string) error
	JSONQueryCF(cf, field, value string) (map[string]string, error)              // Query by JSON field
	SearchCF(cf string, opts SearchOptions) (*SearchResults, error)              // Fuzzy search in column family
	ExportSearchResultsToCSV(cf, filePath, sep string, opts SearchOptions) error // Export search results to CSV
	ListCFs() ([]string, error)
	CreateCF(cf string) error
	DropCF(cf string) error
	GetCFStats(cf string) (*CFStats, error)    // Get statistics for a specific column family
	GetDatabaseStats() (*DatabaseStats, error) // Get overall database statistics
	IsReadOnly() bool
	Close()

	// Smart key conversion methods
	SmartGetCF(cf, key string) (string, error)
	SmartPrefixScanCF(cf, prefix string, limit int) (map[string]string, error)
	SmartScanCF(cf string, start, end string, opts ScanOptions) (map[string]string, error)
	SmartScanCFPage(cf string, start, end string, opts ScanOptions) (ScanPageResult, error) // new paginated version
	GetKeyFormatInfo(cf string) (util.KeyFormat, string)
}

type DB struct {
	db         *grocksdb.DB
	cfHandles  map[string]*grocksdb.ColumnFamilyHandle
	ro         *grocksdb.ReadOptions
	wo         *grocksdb.WriteOptions
	readOnly   bool
	keyFormats map[string]util.KeyFormat // Cache of detected key formats per CF
	formatMux  sync.RWMutex              // Mutex for keyFormats map
}

func Open(path string) (*DB, error) {
	return OpenWithOptions(path, false)
}

func OpenReadOnly(path string) (*DB, error) {
	return OpenWithOptions(path, true)
}

func OpenWithOptions(path string, readOnly bool) (*DB, error) {
	cfNames, err := grocksdb.ListColumnFamilies(grocksdb.NewDefaultOptions(), path)
	if err != nil || len(cfNames) == 0 {
		cfNames = []string{"default"}
	}
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)
	cfOpts := make([]*grocksdb.Options, len(cfNames))
	for i := range cfNames {
		cfOpts[i] = grocksdb.NewDefaultOptions()
	}

	var db *grocksdb.DB
	var cfHandles []*grocksdb.ColumnFamilyHandle

	if readOnly {
		// Use read-only mode - don't create missing column families in read-only mode
		opts.SetCreateIfMissing(false)
		opts.SetCreateIfMissingColumnFamilies(false)
		db, cfHandles, err = grocksdb.OpenDbForReadOnlyColumnFamilies(opts, path, cfNames, cfOpts, false)
	} else {
		db, cfHandles, err = grocksdb.OpenDbColumnFamilies(opts, path, cfNames, cfOpts)
	}

	if err != nil {
		return nil, err
	}
	cfHandleMap := make(map[string]*grocksdb.ColumnFamilyHandle)
	for i, name := range cfNames {
		cfHandleMap[name] = cfHandles[i]
	}
	return &DB{
		db:         db,
		cfHandles:  cfHandleMap,
		ro:         grocksdb.NewDefaultReadOptions(),
		wo:         grocksdb.NewDefaultWriteOptions(),
		readOnly:   readOnly,
		keyFormats: make(map[string]util.KeyFormat),
		formatMux:  sync.RWMutex{},
	}, nil
}

func (d *DB) Close() {
	for _, h := range d.cfHandles {
		h.Destroy()
	}
	d.db.Close()
	d.ro.Destroy()
	d.wo.Destroy()
}

func (d *DB) GetCF(cf, key string) (string, error) {
	h, ok := d.cfHandles[cf]
	if !ok {
		return "", ErrColumnFamilyNotFound
	}
	val, err := d.db.GetCF(d.ro, h, []byte(key))
	if err != nil {
		return "", err
	}
	defer val.Free()
	if !val.Exists() {
		return "", ErrKeyNotFound
	}
	return string(val.Data()), nil
}

func (d *DB) PutCF(cf, key, value string) error {
	if d.readOnly {
		return ErrReadOnlyMode
	}
	h, ok := d.cfHandles[cf]
	if !ok {
		return ErrColumnFamilyNotFound
	}
	return d.db.PutCF(d.wo, h, []byte(key), []byte(value))
}

func (d *DB) PrefixScanCF(cf, prefix string, limit int) (map[string]string, error) {
	h, ok := d.cfHandles[cf]
	if !ok {
		return nil, ErrColumnFamilyNotFound
	}
	it := d.db.NewIteratorCF(d.ro, h)
	defer it.Close()
	result := make(map[string]string)
	for it.Seek([]byte(prefix)); it.Valid(); it.Next() {
		k := it.Key()
		v := it.Value()
		if !hasPrefix(k.Data(), []byte(prefix)) {
			k.Free()
			v.Free()
			break
		}
		result[string(k.Data())] = string(v.Data())
		k.Free()
		v.Free()
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (d *DB) ScanCF(cf string, start, end []byte, opts ScanOptions) (map[string]string, error) {
	h, ok := d.cfHandles[cf]
	if !ok {
		return nil, ErrColumnFamilyNotFound
	}

	it := d.db.NewIteratorCF(d.ro, h)
	defer it.Close()

	result := make(map[string]string)
	startStr := string(start)
	endStr := string(end)

	// Position iterator based on direction and bounds
	if opts.Reverse {
		// For reverse scan, we start from end and go backwards to start
		if len(end) > 0 {
			it.SeekForPrev(end)
		} else if len(start) > 0 {
			// Fix: when only start is specified, start from start key, not last record
			it.SeekForPrev(start)
		} else {
			it.SeekToLast()
		}
	} else {
		// For forward scan, we start from start and go forwards to end
		if len(start) > 0 {
			it.Seek(start)
		} else {
			it.SeekToFirst()
		}
	}

	// Iterate over the range
	for it.Valid() {
		k := it.Key()
		kStr := string(k.Data())

		// Check bounds based on direction
		if opts.Reverse {
			// For reverse scan: stop when we reach below start (only if end is also specified)
			if len(start) > 0 && len(end) > 0 && kStr < startStr {
				k.Free()
				break
			}
			// For reverse scan: skip if we're at or above end
			if len(end) > 0 && kStr >= endStr {
				k.Free()
				it.Prev()
				continue
			}
		} else {
			// For forward scan: stop when we reach end
			if len(end) > 0 && kStr >= endStr {
				k.Free()
				break
			}
			// For forward scan: skip if we're below start
			if len(start) > 0 && kStr < startStr {
				k.Free()
				it.Next()
				continue
			}
		}

		// Store key-value pair
		if opts.Values {
			v := it.Value()
			result[kStr] = string(v.Data())
			v.Free()
		} else {
			result[kStr] = ""
		}
		k.Free()

		// Check limit
		if opts.Limit > 0 && len(result) >= opts.Limit {
			break
		}

		// Move iterator
		if opts.Reverse {
			it.Prev()
		} else {
			it.Next()
		}
	}

	return result, nil
}

func (d *DB) ListCFs() ([]string, error) {
	return grocksdb.ListColumnFamilies(grocksdb.NewDefaultOptions(), d.db.Name())
}

func (d *DB) CreateCF(cf string) error {
	if d.readOnly {
		return ErrReadOnlyMode
	}
	// Check if column family already exists
	if _, exists := d.cfHandles[cf]; exists {
		return ErrColumnFamilyExists
	}
	h, err := d.db.CreateColumnFamily(grocksdb.NewDefaultOptions(), cf)
	if err != nil {
		return err
	}
	d.cfHandles[cf] = h
	return nil
}

func (d *DB) DropCF(cf string) error {
	if d.readOnly {
		return ErrReadOnlyMode
	}
	h, ok := d.cfHandles[cf]
	if !ok {
		return ErrColumnFamilyNotFound
	}
	err := d.db.DropColumnFamily(h)
	if err != nil {
		return err
	}
	h.Destroy()
	delete(d.cfHandles, cf)
	return nil
}

func (d *DB) GetLastCF(cf string) (string, string, error) {
	h, ok := d.cfHandles[cf]
	if !ok {
		return "", "", ErrColumnFamilyNotFound
	}

	it := d.db.NewIteratorCF(d.ro, h)
	defer it.Close()

	// Seek to the last key-value pair
	it.SeekToLast()
	if !it.Valid() {
		return "", "", ErrColumnFamilyEmpty
	}

	k := it.Key()
	v := it.Value()
	defer k.Free()
	defer v.Free()

	return string(k.Data()), string(v.Data()), nil
}

func (d *DB) ExportToCSV(cf, filePath, sep string) error {
	h, ok := d.cfHandles[cf]
	if !ok {
		return ErrColumnFamilyNotFound
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if sep == "" {
		sep = ","
	}
	runes := []rune(sep)
	if len(runes) != 1 {
		return fmt.Errorf("CSV separator must be a single character, got: %q", sep)
	}
	writer.Comma = runes[0]
	defer writer.Flush()

	// Write CSV header
	err = writer.Write([]string{"Key", "Value"})
	if err != nil {
		return err
	}

	it := d.db.NewIteratorCF(d.ro, h)
	defer it.Close()

	for it.SeekToFirst(); it.Valid(); it.Next() {
		k := it.Key()
		v := it.Value()

		err := writer.Write([]string{util.FormatKey(string(k.Data())), string(v.Data())})
		if err != nil {
			k.Free()
			v.Free()
			return err
		}

		k.Free()
		v.Free()
	}

	return nil
}

// ExportSearchResultsToCSV exports search results to a CSV file
func (d *DB) ExportSearchResultsToCSV(cf, filePath, sep string, opts SearchOptions) error {
	// First, perform the search to get results
	results, err := d.SearchCF(cf, opts)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if sep == "" {
		sep = ","
	}
	runes := []rune(sep)
	if len(runes) != 1 {
		return fmt.Errorf("CSV separator must be a single character, got: %q", sep)
	}
	writer.Comma = runes[0]
	defer writer.Flush()

	// Write CSV header
	if opts.KeysOnly {
		err = writer.Write([]string{"Key"})
	} else {
		err = writer.Write([]string{"Key", "Value"})
	}
	if err != nil {
		return err
	}

	// Write search results
	for _, result := range results.Results {
		if opts.KeysOnly {
			err := writer.Write([]string{util.FormatKey(result.Key)})
			if err != nil {
				return err
			}
		} else {
			err := writer.Write([]string{util.FormatKey(result.Key), result.Value})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func hasPrefix(s, prefix []byte) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := range prefix {
		if s[i] != prefix[i] {
			return false
		}
	}
	return true
}

func (d *DB) JSONQueryCF(cf, field, value string) (map[string]string, error) {
	h, ok := d.cfHandles[cf]
	if !ok {
		return nil, ErrColumnFamilyNotFound
	}

	it := d.db.NewIteratorCF(d.ro, h)
	defer it.Close()

	result := make(map[string]string)

	for it.SeekToFirst(); it.Valid(); it.Next() {
		k := it.Key()
		v := it.Value()
		keyStr := string(k.Data())
		valueStr := string(v.Data())

		// Try to parse as JSON
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(valueStr), &jsonData); err != nil {
			// Skip non-JSON values
			k.Free()
			v.Free()
			continue
		}

		// Check if the field exists and matches the value
		if fieldValue, exists := jsonData[field]; exists {
			var match bool

			// Handle different value types
			switch v := fieldValue.(type) {
			case string:
				match = v == value
			case float64:
				// Try to parse the input value as number
				if numValue, err := strconv.ParseFloat(value, 64); err == nil {
					match = v == numValue
				}
			case bool:
				// Try to parse the input value as boolean
				if boolValue, err := strconv.ParseBool(value); err == nil {
					match = v == boolValue
				}
			case nil:
				match = value == "null"
			default:
				// For other types, convert to string and compare
				fieldValueStr := json.RawMessage(fmt.Sprintf("%v", v))
				var prettyFieldValue string
				if err := json.Unmarshal(fieldValueStr, &prettyFieldValue); err == nil {
					match = prettyFieldValue == value
				}
			}

			if match {
				result[keyStr] = valueStr
			}
		}

		k.Free()
		v.Free()
	}

	return result, nil
}

func (d *DB) IsReadOnly() bool {
	return d.readOnly
}

// detectDataType analyzes a value and determines its most likely data type
func detectDataType(value string) DataType {
	if len(value) == 0 {
		return DataTypeEmpty
	}

	// Check for JSON
	trimmed := strings.TrimSpace(value)
	if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
		(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
		var jsonData interface{}
		if json.Unmarshal([]byte(value), &jsonData) == nil {
			return DataTypeJSON
		}
	}

	// Check for numbers
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return DataTypeNumber
	}

	// Check for timestamps (Unix timestamps in various formats)
	if ts, err := strconv.ParseInt(value, 10, 64); err == nil {
		// Reasonable timestamp range (covers years 1973-2033 for seconds, and microseconds/nanoseconds)
		if (ts > 1e8 && ts < 2e9) || (ts > 1e12 && ts < 2e18) {
			return DataTypeTimestamp
		}
	}

	// Check for binary data (contains non-printable characters)
	for _, b := range []byte(value) {
		if b < 32 && b != 9 && b != 10 && b != 13 { // Allow tab, newline, carriage return
			return DataTypeBinary
		}
	}

	return DataTypeString
}

// getKeyPrefix extracts a meaningful prefix from a key for common prefix analysis
func getKeyPrefix(key string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 10 // Default prefix length
	}

	// Look for common separators
	separators := []string{":", "/", "-", "_", "."}
	for _, sep := range separators {
		if idx := strings.Index(key, sep); idx > 0 && idx <= maxLen {
			return key[:idx+1] // Include the separator
		}
	}

	// If no separator found, use first maxLen characters
	if len(key) <= maxLen {
		return key
	}
	return key[:maxLen]
}

func (d *DB) GetCFStats(cf string) (*CFStats, error) {
	h, ok := d.cfHandles[cf]
	if !ok {
		return nil, ErrColumnFamilyNotFound
	}

	stats := &CFStats{
		Name:                    cf,
		DataTypeDistribution:    make(map[DataType]int64),
		KeyLengthDistribution:   make(map[string]int64),
		ValueLengthDistribution: make(map[string]int64),
		CommonPrefixes:          make(map[string]int64),
		SampleKeys:              make([]string, 0, 10),
		LastUpdated:             time.Now(),
	}

	it := d.db.NewIteratorCF(d.ro, h)
	defer it.Close()

	sampleCount := 0
	const maxSamples = 10

	for it.SeekToFirst(); it.Valid(); it.Next() {
		k := it.Key()
		v := it.Value()

		keyStr := string(k.Data())
		valueStr := string(v.Data())

		// Update counters
		stats.KeyCount++
		keyLen := int64(len(keyStr))
		valueLen := int64(len(valueStr))
		stats.TotalKeySize += keyLen
		stats.TotalValueSize += valueLen

		// Detect data type
		dataType := detectDataType(valueStr)
		stats.DataTypeDistribution[dataType]++

		// Key length distribution (categorized)
		keyLenCategory := categorizeLength(keyLen)
		stats.KeyLengthDistribution[keyLenCategory]++

		// Value length distribution (categorized)
		valueLenCategory := categorizeLength(valueLen)
		stats.ValueLengthDistribution[valueLenCategory]++

		// Common prefixes analysis
		prefix := getKeyPrefix(keyStr, 10)
		stats.CommonPrefixes[prefix]++

		// Collect sample keys
		if sampleCount < maxSamples {
			stats.SampleKeys = append(stats.SampleKeys, keyStr)
			sampleCount++
		}

		k.Free()
		v.Free()
	}

	// Calculate averages
	if stats.KeyCount > 0 {
		stats.AverageKeySize = float64(stats.TotalKeySize) / float64(stats.KeyCount)
		stats.AverageValueSize = float64(stats.TotalValueSize) / float64(stats.KeyCount)
	}

	return stats, nil
}

// categorizeLength converts a byte length into a human-readable category
func categorizeLength(length int64) string {
	switch {
	case length == 0:
		return "empty"
	case length <= 10:
		return "tiny (≤10)"
	case length <= 100:
		return "small (11-100)"
	case length <= 1000:
		return "medium (101-1K)"
	case length <= 10000:
		return "large (1K-10K)"
	case length <= 100000:
		return "very large (10K-100K)"
	default:
		return "huge (>100K)"
	}
}

func (d *DB) GetDatabaseStats() (*DatabaseStats, error) {
	cfs, err := d.ListCFs()
	if err != nil {
		return nil, err
	}

	stats := &DatabaseStats{
		ColumnFamilies:    make([]CFStats, 0, len(cfs)),
		ColumnFamilyCount: len(cfs),
		LastUpdated:       time.Now(),
	}

	for _, cf := range cfs {
		cfStats, err := d.GetCFStats(cf)
		if err != nil {
			// Continue with other CFs even if one fails
			continue
		}

		stats.ColumnFamilies = append(stats.ColumnFamilies, *cfStats)
		stats.TotalKeyCount += cfStats.KeyCount
		stats.TotalSize += cfStats.TotalKeySize + cfStats.TotalValueSize
	}

	return stats, nil
}

// SearchCF performs fuzzy search in a column family based on provided options
func (d *DB) SearchCF(cf string, opts SearchOptions) (*SearchResults, error) {
	startTime := time.Now()

	h, ok := d.cfHandles[cf]
	if !ok {
		return nil, ErrColumnFamilyNotFound
	}

	it := d.db.NewIteratorCF(d.ro, h)
	defer it.Close()

	results := &SearchResults{
		Results: make([]SearchResult, 0),
		Limited: false,
	}

	// Compile regex patterns if needed
	var keyRegex, valueRegex *regexp.Regexp
	var err error

	if opts.UseRegex {
		if opts.KeyPattern != "" {
			flags := ""
			if !opts.CaseSensitive {
				flags = "(?i)"
			}
			keyRegex, err = regexp.Compile(flags + opts.KeyPattern)
			if err != nil {
				return nil, fmt.Errorf("invalid key regex pattern: %v", err)
			}
		}
		if opts.ValuePattern != "" {
			flags := ""
			if !opts.CaseSensitive {
				flags = "(?i)"
			}
			valueRegex, err = regexp.Compile(flags + opts.ValuePattern)
			if err != nil {
				return nil, fmt.Errorf("invalid value regex pattern: %v", err)
			}
		}
	}

	// 游标分页：先跳过所有 key <= After
	foundAfter := opts.After == ""
	var afterKeyBytes []byte
	var useByteComparison bool

	// If After is specified, try to convert it to the appropriate binary format
	if !foundAfter {
		keyFormat := d.getKeyFormat(cf)
		if keyFormat == util.KeyFormatUint64BE {
			// For numeric keys, convert the after string to binary format
			if convertedKey, err := util.ConvertStringToKey(opts.After, keyFormat); err == nil {
				afterKeyBytes = convertedKey
				useByteComparison = true
			}
		}
	}

	var lastKey string
	for it.SeekToFirst(); it.Valid(); it.Next() {
		k := it.Key()
		keyStr := string(k.Data())
		keyBytes := k.Data()

		if !foundAfter {
			var shouldSkip bool
			if useByteComparison && afterKeyBytes != nil {
				// Use byte comparison for numeric keys
				shouldSkip = compareBytes(keyBytes, afterKeyBytes) <= 0
			} else {
				// Use string comparison for string keys or fallback
				shouldSkip = keyStr <= opts.After
			}

			if !shouldSkip {
				foundAfter = true
			} else {
				k.Free()
				continue
			}
		}

		v := it.Value()
		valueStr := string(v.Data())

		var keyMatches, valueMatches bool
		var matchedFields []string

		// Check key pattern matching
		if opts.KeyPattern != "" {
			keyMatches = matchPattern(keyStr, opts.KeyPattern, opts.UseRegex, opts.CaseSensitive, keyRegex)
			if keyMatches {
				matchedFields = append(matchedFields, "key")
			}
		}

		// Check value pattern matching
		if opts.ValuePattern != "" {
			valueMatches = matchPattern(valueStr, opts.ValuePattern, opts.UseRegex, opts.CaseSensitive, valueRegex)
			if valueMatches {
				matchedFields = append(matchedFields, "value")
			}
		}

		// Determine if this entry should be included in results
		shouldInclude := false
		if opts.KeyPattern != "" && opts.ValuePattern != "" {
			shouldInclude = keyMatches && valueMatches
		} else if opts.KeyPattern != "" {
			shouldInclude = keyMatches
		} else if opts.ValuePattern != "" {
			shouldInclude = valueMatches
		}

		if shouldInclude {
			displayKey := keyStr
			if opts.Tick {
				// Convert key to UTC time string if Tick option is enabled
				displayKey = convertTickTimeToUTC(keyStr)
			}

			result := SearchResult{
				Key:           displayKey,
				MatchedFields: matchedFields,
			}
			if !opts.KeysOnly {
				result.Value = valueStr
			}
			results.Results = append(results.Results, result)
			lastKey = keyStr
			if opts.Limit > 0 && len(results.Results) >= opts.Limit {
				break
			}
		}
		k.Free()
		v.Free()
	}

	results.Total = len(results.Results)
	results.QueryTime = time.Since(startTime).String()
	results.NextCursor = ""
	results.HasMore = false
	if opts.Limit > 0 && len(results.Results) >= opts.Limit && it.Valid() {
		results.NextCursor = lastKey
		results.HasMore = true
	}

	return results, nil
}

// matchPattern checks if text matches the given pattern
func matchPattern(text, pattern string, useRegex, caseSensitive bool, compiledRegex *regexp.Regexp) bool {
	if pattern == "" {
		return true
	}

	if useRegex {
		if compiledRegex != nil {
			return compiledRegex.MatchString(text)
		}
		return false
	}

	// Handle wildcard patterns (* and ?)
	if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") {
		return matchWildcard(text, pattern, caseSensitive)
	}

	// Simple substring matching
	if caseSensitive {
		return strings.Contains(text, pattern)
	}
	return strings.Contains(strings.ToLower(text), strings.ToLower(pattern))
}

// matchWildcard performs wildcard pattern matching (* for any sequence, ? for single character)
func matchWildcard(text, pattern string, caseSensitive bool) bool {
	if !caseSensitive {
		text = strings.ToLower(text)
		pattern = strings.ToLower(pattern)
	}

	return wildcardMatch(text, pattern, 0, 0)
}

// wildcardMatch is a recursive function for wildcard pattern matching
func wildcardMatch(text, pattern string, textIdx, patternIdx int) bool {
	// Base cases
	if patternIdx >= len(pattern) {
		return textIdx >= len(text)
	}
	if textIdx >= len(text) {
		// Check if remaining pattern consists only of '*'
		for i := patternIdx; i < len(pattern); i++ {
			if pattern[i] != '*' {
				return false
			}
		}
		return true
	}

	currentChar := pattern[patternIdx]

	switch currentChar {
	case '*':
		// Try matching zero or more characters
		// First try matching zero characters (skip the *)
		if wildcardMatch(text, pattern, textIdx, patternIdx+1) {
			return true
		}
		// Then try matching one character and continue
		return wildcardMatch(text, pattern, textIdx+1, patternIdx)

	case '?':
		// Match exactly one character
		return wildcardMatch(text, pattern, textIdx+1, patternIdx+1)

	default:
		// Match exact character
		if text[textIdx] == currentChar {
			return wildcardMatch(text, pattern, textIdx+1, patternIdx+1)
		}
		return false
	}
}

// getKeyFormat returns the cached key format for a column family, detecting it if needed
func (d *DB) getKeyFormat(cf string) util.KeyFormat {
	d.formatMux.RLock()
	if format, exists := d.keyFormats[cf]; exists {
		d.formatMux.RUnlock()
		return format
	}
	d.formatMux.RUnlock()

	// Format not cached, detect it
	format := d.detectKeyFormat(cf)

	d.formatMux.Lock()
	if d.keyFormats == nil {
		d.keyFormats = make(map[string]util.KeyFormat)
	}
	d.keyFormats[cf] = format
	d.formatMux.Unlock()

	return format
}

// detectKeyFormat analyzes keys in a column family to determine their format
func (d *DB) detectKeyFormat(cf string) util.KeyFormat {
	h, ok := d.cfHandles[cf]
	if !ok {
		return util.KeyFormatString
	}

	it := d.db.NewIteratorCF(d.ro, h)
	defer it.Close()

	var sampleKeys []string
	sampleSize := 0
	maxSamples := 20 // Sample up to 20 keys for format detection

	// Collect sample keys
	for it.SeekToFirst(); it.Valid() && sampleSize < maxSamples; it.Next() {
		k := it.Key()
		sampleKeys = append(sampleKeys, string(k.Data()))
		k.Free()
		sampleSize++
	}

	return util.DetectKeyFormat(sampleKeys)
}

// InvalidateKeyFormatCache clears the cached key format for a column family
func (d *DB) InvalidateKeyFormatCache(cf string) {
	d.formatMux.Lock()
	if d.keyFormats != nil {
		delete(d.keyFormats, cf)
	}
	d.formatMux.Unlock()
}

// SmartGetCF gets a value by key, automatically converting string input to appropriate binary format
func (d *DB) SmartGetCF(cf, key string) (string, error) {
	// Get the key format for this column family
	format := d.getKeyFormat(cf)

	// Convert string input to appropriate binary key
	binaryKey, err := util.ConvertStringToKey(key, format)
	if err != nil {
		// If conversion fails, fall back to original string key
		binaryKey = []byte(key)
	}

	h, ok := d.cfHandles[cf]
	if !ok {
		return "", ErrColumnFamilyNotFound
	}

	val, err := d.db.GetCF(d.ro, h, binaryKey)
	if err != nil {
		return "", err
	}
	defer val.Free()
	if !val.Exists() {
		return "", ErrKeyNotFound
	}
	return string(val.Data()), nil
}

// SmartPrefixScanCF performs prefix scan with automatic key conversion
func (d *DB) SmartPrefixScanCF(cf, prefix string, limit int) (map[string]string, error) {
	// Get the key format for this column family
	format := d.getKeyFormat(cf)

	// Convert string prefix to appropriate binary format
	binaryPrefix, err := util.ConvertStringToKeyForScan(prefix, format, true)
	if err != nil {
		// If conversion fails, fall back to original string prefix
		binaryPrefix = []byte(prefix)
	}

	h, ok := d.cfHandles[cf]
	if !ok {
		return nil, ErrColumnFamilyNotFound
	}

	it := d.db.NewIteratorCF(d.ro, h)
	defer it.Close()
	result := make(map[string]string)

	for it.Seek(binaryPrefix); it.Valid(); it.Next() {
		k := it.Key()
		v := it.Value()
		if !hasPrefix(k.Data(), binaryPrefix) {
			k.Free()
			v.Free()
			break
		}
		result[string(k.Data())] = string(v.Data())
		k.Free()
		v.Free()
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result, nil
}

// SmartScanCF performs range scan with automatic key conversion
func (d *DB) SmartScanCF(cf string, start, end string, opts ScanOptions) (map[string]string, error) {
	// Get the key format for this column family
	format := d.getKeyFormat(cf)

	// Convert string bounds to appropriate binary format
	var startBytes, endBytes []byte
	var err error

	if start != "" && start != "*" {
		startBytes, err = util.ConvertStringToKeyForScan(start, format, false)
		if err != nil {
			startBytes = []byte(start) // Fall back to string
		}
	}

	if end != "" && end != "*" {
		endBytes, err = util.ConvertStringToKeyForScan(end, format, false)
		if err != nil {
			endBytes = []byte(end) // Fall back to string
		}
	}

	return d.ScanCF(cf, startBytes, endBytes, opts)
}

// GetKeyFormatInfo returns information about the detected key format for a column family
func (d *DB) GetKeyFormatInfo(cf string) (util.KeyFormat, string) {
	format := d.getKeyFormat(cf)
	var description string

	switch format {
	case util.KeyFormatUint64BE:
		description = "8-byte big-endian unsigned integers"
	case util.KeyFormatHex:
		description = "Hexadecimal-encoded binary data"
	case util.KeyFormatMixed:
		description = "Mixed format (binary and string keys)"
	case util.KeyFormatString:
		description = "Printable string keys"
	default:
		description = "Unknown format"
	}

	return format, description
}

// ScanCFPage implements cursor-based pagination
func (d *DB) ScanCFPage(cf string, start, end []byte, opts ScanOptions) (ScanPageResult, error) {
	h, ok := d.cfHandles[cf]
	if !ok {
		return ScanPageResult{}, ErrColumnFamilyNotFound
	}

	it := d.db.NewIteratorCF(d.ro, h)
	defer it.Close()

	result := make(map[string]string)
	startStr := string(start)
	endStr := string(end)
	startAfter := opts.StartAfter
	var lastKey string
	count := 0

	if opts.Reverse {
		if len(end) > 0 {
			it.SeekForPrev(end)
		} else if len(start) > 0 {
			it.SeekForPrev(start)
		} else {
			it.SeekToLast()
		}
		// Advance until key < startAfter
		if startAfter != "" {
			for it.Valid() {
				k := it.Key()
				kStr := string(k.Data())
				k.Free()
				if kStr < startAfter {
					break
				}
				it.Prev()
			}
		}
	} else {
		if len(start) > 0 {
			it.Seek(start)
		} else {
			it.SeekToFirst()
		}
		// Advance until key > startAfter
		if startAfter != "" {
			for it.Valid() {
				k := it.Key()
				kStr := string(k.Data())
				k.Free()
				if kStr > startAfter {
					break
				}
				it.Next()
			}
		}
	}

	// If iterator is invalid after skipping, return empty result
	if !it.Valid() {
		return ScanPageResult{Results: map[string]string{}, NextCursor: "", HasMore: false}, nil
	}

	for it.Valid() {
		k := it.Key()
		kStr := string(k.Data())

		// Check bounds
		if opts.Reverse {
			if len(start) > 0 && len(end) > 0 && kStr < startStr {
				k.Free()
				break
			}
			if len(end) > 0 && kStr >= endStr {
				k.Free()
				it.Prev()
				continue
			}
		} else {
			if len(end) > 0 && kStr >= endStr {
				k.Free()
				break
			}
			if len(start) > 0 && kStr < startStr {
				k.Free()
				it.Next()
				continue
			}
		}

		if opts.Values {
			v := it.Value()
			result[kStr] = string(v.Data())
			v.Free()
		} else {
			result[kStr] = ""
		}
		lastKey = kStr
		count++
		k.Free()

		if opts.Limit > 0 && count >= opts.Limit {
			break
		}

		if opts.Reverse {
			it.Prev()
		} else {
			it.Next()
		}
	}

	hasMore := false
	nextCursor := ""
	if opts.Limit > 0 && count >= opts.Limit {
		if it.Valid() {
			hasMore = true
			nextCursor = lastKey
		}
	}

	return ScanPageResult{
		Results:    result,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

// SmartScanCFPage: like SmartScanCF, but paginated
func (d *DB) SmartScanCFPage(cf string, start, end string, opts ScanOptions) (ScanPageResult, error) {
	format := d.getKeyFormat(cf)
	var startBytes, endBytes []byte
	var err error
	if start != "" && start != "*" {
		startBytes, err = util.ConvertStringToKeyForScan(start, format, false)
		if err != nil {
			startBytes = []byte(start)
		}
	}
	if end != "" && end != "*" {
		endBytes, err = util.ConvertStringToKeyForScan(end, format, false)
		if err != nil {
			endBytes = []byte(end)
		}
	}
	return d.ScanCFPage(cf, startBytes, endBytes, opts)
}

// compareBytes compares two byte slices lexicographically
// Returns -1 if a < b, 0 if a == b, 1 if a > b
func compareBytes(a, b []byte) int {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	for i := 0; i < minLen; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}

	// If all compared bytes are equal, compare lengths
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}

	return 0
}

// convertTickTimeToUTC converts a key that represents a .NET tick time to UTC string
// In .NET, 1 tick = 100 nanoseconds, and 1 second = 10,000,000 ticks
func convertTickTimeToUTC(key string) string {
	// Try to parse as a number (.NET ticks)
	if ticks, err := strconv.ParseInt(key, 10, 64); err == nil {
		// .NET ticks: 1 tick = 100 nanoseconds, 1 second = 10,000,000 ticks
		const ticksPerSecond = 10000000 // 10^7

		// Check if this looks like a reasonable .NET tick value
		// .NET DateTime.Ticks from year 1 to year 9999 would be in this range
		// But we'll focus on modern timestamps (after 1970)
		minTicks := int64(621355968000000000)  // 1970-01-01 00:00:00 UTC in .NET ticks
		maxTicks := int64(3155378975999999999) // 9999-12-31 23:59:59 UTC in .NET ticks

		if ticks >= minTicks && ticks <= maxTicks {
			// Convert .NET ticks to Unix timestamp
			// .NET epoch is 0001-01-01, Unix epoch is 1970-01-01
			// The difference is 621355968000000000 ticks
			unixTicks := ticks - minTicks
			seconds := unixTicks / ticksPerSecond
			nanoseconds := (unixTicks % ticksPerSecond) * 100 // Each tick is 100 nanoseconds

			t := time.Unix(seconds, nanoseconds)
			return t.UTC().Format("2006-01-02 15:04:05.000000 UTC")
		}

		// If not in the valid .NET tick range, try as a simpler timestamp
		// Could be ticks since Unix epoch
		if ticks > ticksPerSecond { // At least 1 second worth of ticks
			seconds := ticks / ticksPerSecond
			nanoseconds := (ticks % ticksPerSecond) * 100

			// Check if this gives us a reasonable date (after 1970)
			if seconds > 0 && seconds < 4102444800 { // Before year 2100
				t := time.Unix(seconds, nanoseconds)
				return t.UTC().Format("2006-01-02 15:04:05.000000 UTC")
			}
		}

		// Not a valid tick timestamp, return original key
		return key
	}

	// If it's binary data, try to interpret as binary .NET ticks
	keyBytes := []byte(key)

	// Try 8-byte big-endian .NET ticks (common in .NET serialization)
	if len(keyBytes) == 8 {
		ticks := int64(binary.BigEndian.Uint64(keyBytes))
		if ticks > 0 {
			const ticksPerSecond = 10000000 // 10^7

			// Check for .NET DateTime.Ticks format
			minTicks := int64(621355968000000000)  // 1970-01-01 00:00:00 UTC in .NET ticks
			maxTicks := int64(3155378975999999999) // 9999-12-31 23:59:59 UTC in .NET ticks

			if ticks >= minTicks && ticks <= maxTicks {
				// Convert .NET ticks to Unix timestamp
				unixTicks := ticks - minTicks
				seconds := unixTicks / ticksPerSecond
				nanoseconds := (unixTicks % ticksPerSecond) * 100

				t := time.Unix(seconds, nanoseconds)
				return t.UTC().Format("2006-01-02 15:04:05.000000 UTC")
			}

			// Try as ticks since Unix epoch
			if ticks > ticksPerSecond {
				seconds := ticks / ticksPerSecond
				nanoseconds := (ticks % ticksPerSecond) * 100

				if seconds > 0 && seconds < 4102444800 { // Before year 2100
					t := time.Unix(seconds, nanoseconds)
					return t.UTC().Format("2006-01-02 15:04:05.000000 UTC")
				}
			}
		}
	}

	// Try 4-byte big-endian as seconds (fallback for non-tick data)
	if len(keyBytes) == 4 {
		seconds := int64(binary.BigEndian.Uint32(keyBytes))
		if seconds > 1e8 { // Reasonable timestamp range (> ~1973)
			t := time.Unix(seconds, 0)
			return t.UTC().Format("2006-01-02 15:04:05.000000 UTC")
		}
	}

	// If all parsing attempts fail, return original key
	return key
}
