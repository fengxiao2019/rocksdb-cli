package service

import (
	"rocksdb-cli/internal/db"
)

// StatsService provides database statistics operations
type StatsService struct {
	db db.KeyValueDB
}

// DatabaseStats contains overall database statistics
type DatabaseStats struct {
	ColumnFamilies    []ColumnFamilyStats `json:"column_families"`
	TotalKeyCount     int64               `json:"total_key_count"`
	TotalSize         int64               `json:"total_size"`
	ColumnFamilyCount int                 `json:"column_family_count"`
}

// ColumnFamilyStats contains statistics for a single column family
type ColumnFamilyStats struct {
	Name                    string                 `json:"name"`
	KeyCount                int64                  `json:"key_count"`
	TotalKeySize            int64                  `json:"total_key_size"`
	TotalValueSize          int64                  `json:"total_value_size"`
	AverageKeySize          float64                `json:"average_key_size"`
	AverageValueSize        float64                `json:"average_value_size"`
	DataTypeDistribution    map[string]int64       `json:"data_type_distribution"`
	KeyLengthDistribution   map[string]int64       `json:"key_length_distribution"`
	ValueLengthDistribution map[string]int64       `json:"value_length_distribution"`
	CommonPrefixes          map[string]int64       `json:"common_prefixes"`
	SampleKeys              []string               `json:"sample_keys"`
}

// NewStatsService creates a new StatsService instance
func NewStatsService(database db.KeyValueDB) *StatsService {
	return &StatsService{db: database}
}

// GetDatabaseStats retrieves overall database statistics
func (s *StatsService) GetDatabaseStats() (*DatabaseStats, error) {
	dbStats, err := s.db.GetDatabaseStats()
	if err != nil {
		return nil, err
	}

	// Convert db stats to service stats
	cfStats := make([]ColumnFamilyStats, 0, len(dbStats.ColumnFamilies))
	for _, cf := range dbStats.ColumnFamilies {
		// Convert DataType enum to string for JSON
		dataTypeDist := make(map[string]int64)
		for dt, count := range cf.DataTypeDistribution {
			dataTypeDist[string(dt)] = count
		}

		cfStats = append(cfStats, ColumnFamilyStats{
			Name:                    cf.Name,
			KeyCount:                cf.KeyCount,
			TotalKeySize:            cf.TotalKeySize,
			TotalValueSize:          cf.TotalValueSize,
			AverageKeySize:          cf.AverageKeySize,
			AverageValueSize:        cf.AverageValueSize,
			DataTypeDistribution:    dataTypeDist,
			KeyLengthDistribution:   cf.KeyLengthDistribution,
			ValueLengthDistribution: cf.ValueLengthDistribution,
			CommonPrefixes:          cf.CommonPrefixes,
			SampleKeys:              cf.SampleKeys,
		})
	}

	return &DatabaseStats{
		ColumnFamilies:    cfStats,
		TotalKeyCount:     dbStats.TotalKeyCount,
		TotalSize:         dbStats.TotalSize,
		ColumnFamilyCount: dbStats.ColumnFamilyCount,
	}, nil
}

// GetColumnFamilyStats retrieves statistics for a specific column family
func (s *StatsService) GetColumnFamilyStats(cf string) (*ColumnFamilyStats, error) {
	cfStats, err := s.db.GetCFStats(cf)
	if err != nil {
		return nil, err
	}

	// Convert DataType enum to string for JSON
	dataTypeDist := make(map[string]int64)
	for dt, count := range cfStats.DataTypeDistribution {
		dataTypeDist[string(dt)] = count
	}

	return &ColumnFamilyStats{
		Name:                    cfStats.Name,
		KeyCount:                cfStats.KeyCount,
		TotalKeySize:            cfStats.TotalKeySize,
		TotalValueSize:          cfStats.TotalValueSize,
		AverageKeySize:          cfStats.AverageKeySize,
		AverageValueSize:        cfStats.AverageValueSize,
		DataTypeDistribution:    dataTypeDist,
		KeyLengthDistribution:   cfStats.KeyLengthDistribution,
		ValueLengthDistribution: cfStats.ValueLengthDistribution,
		CommonPrefixes:          cfStats.CommonPrefixes,
		SampleKeys:              cfStats.SampleKeys,
	}, nil
}
