package db

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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

type ScanOptions struct {
	Limit   int
	Reverse bool
	Values  bool
}

type KeyValueDB interface {
	GetCF(cf, key string) (string, error)
	PutCF(cf, key, value string) error
	PrefixScanCF(cf, prefix string, limit int) (map[string]string, error)
	ScanCF(cf string, start, end []byte, opts ScanOptions) (map[string]string, error)
	GetLastCF(cf string) (string, string, error) // Returns key, value, error
	ExportToCSV(cf, filePath string) error
	JSONQueryCF(cf, field, value string) (map[string]string, error) // Query by JSON field
	ListCFs() ([]string, error)
	CreateCF(cf string) error
	DropCF(cf string) error
	GetCFStats(cf string) (*CFStats, error)    // Get statistics for a specific column family
	GetDatabaseStats() (*DatabaseStats, error) // Get overall database statistics
	IsReadOnly() bool
	Close()
}

type DB struct {
	db        *grocksdb.DB
	cfHandles map[string]*grocksdb.ColumnFamilyHandle
	ro        *grocksdb.ReadOptions
	wo        *grocksdb.WriteOptions
	readOnly  bool
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
		db:        db,
		cfHandles: cfHandleMap,
		ro:        grocksdb.NewDefaultReadOptions(),
		wo:        grocksdb.NewDefaultWriteOptions(),
		readOnly:  readOnly,
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

func (d *DB) ExportToCSV(cf, filePath string) error {
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

		err := writer.Write([]string{string(k.Data()), string(v.Data())})
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
