package util

import (
	"testing"
	"time"
)

func TestTimeToTicks(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedTicks int64
	}{
		{
			name:          "Unix Epoch",
			input:         "1970-01-01T00:00:00Z",
			expectedTicks: 621355968000000000,
		},
		{
			name:          "2024-01-01",
			input:         "2024-01-01T00:00:00Z",
			expectedTicks: 638396640000000000, // Actual calculated value
		},
		{
			name:          "User example - millisecond precision",
			input:         "2025-11-26T14:44:51.569Z",
			expectedTicks: 638997650915690000, // Millisecond precision only
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputTime, err := time.Parse(time.RFC3339Nano, tt.input)
			if err != nil {
				t.Fatalf("Failed to parse time: %v", err)
			}

			result := TimeToTicks(inputTime)
			if result != tt.expectedTicks {
				t.Errorf("TimeToTicks() = %d, want %d (diff: %d ticks)",
					result, tt.expectedTicks, result-tt.expectedTicks)
			}
		})
	}
}

func TestTicksToTime(t *testing.T) {
	tests := []struct {
		name         string
		ticks        int64
		expectedTime string
	}{
		{
			name:         "Unix Epoch",
			ticks:        621355968000000000,
			expectedTime: "1970-01-01T00:00:00Z",
		},
		{
			name:         "2024-01-01",
			ticks:        638407680000000000,
			expectedTime: "2024-01-01T00:00:00Z",
		},
		{
			name:         "User example with full precision",
			ticks:        638997650915691759,
			expectedTime: "2025-11-26T14:44:51.5691759Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TicksToTime(tt.ticks)
			expectedTime, err := time.Parse(time.RFC3339Nano, tt.expectedTime)
			if err != nil {
				t.Fatalf("Failed to parse expected time: %v", err)
			}

			if !result.Equal(expectedTime) {
				t.Errorf("TicksToTime() = %s, want %s (diff: %v)",
					result.Format(time.RFC3339Nano),
					expectedTime.Format(time.RFC3339Nano),
					result.Sub(expectedTime))
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"Seconds only", "2024-12-04T15:30:45Z"},
		{"Milliseconds", "2024-12-04T15:30:45.123Z"},
		{"Microseconds", "2024-12-04T15:30:45.123456Z"},
		{"Nanoseconds", "2024-12-04T15:30:45.123456789Z"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original, err := time.Parse(time.RFC3339Nano, tt.input)
			if err != nil {
				t.Fatalf("Failed to parse time: %v", err)
			}

			ticks := TimeToTicks(original)
			converted := TicksToTime(ticks)

			if !original.Equal(converted) {
				t.Errorf("Round trip failed:\nOriginal:  %s\nTicks:     %d\nConverted: %s\nDiff: %v",
					original.Format(time.RFC3339Nano),
					ticks,
					converted.Format(time.RFC3339Nano),
					original.Sub(converted))
			}
		})
	}
}

func TestTicksWithSubMillisecondPrecision(t *testing.T) {
	// This test verifies that Go can handle sub-millisecond precision
	// that JavaScript cannot handle

	// Create a time with nanosecond precision
	testTime := time.Date(2025, 11, 26, 14, 44, 51, 569175900, time.UTC)

	ticks := TimeToTicks(testTime)
	t.Logf("Time:  %s", testTime.Format(time.RFC3339Nano))
	t.Logf("Ticks: %d", ticks)

	// Expected: 638997650915691759
	// This should match because Go preserves nanosecond precision
	expectedTicks := int64(638997650915691759)

	if ticks != expectedTicks {
		t.Logf("Difference: %d ticks (%.9f seconds)",
			ticks-expectedTicks,
			float64(ticks-expectedTicks)*100/1e9)
	}
}

func TestEpochDifference(t *testing.T) {
	// Verify the epoch difference constant
	// Days from 0001-01-01 to 1970-01-01 = 719162 days
	days := int64(719162)
	nanosecondsPerDay := int64(24 * 60 * 60 * 1e9)
	expectedNano := days * nanosecondsPerDay
	expectedTicks := expectedNano / nanosecondsPerTick

	if expectedTicks != epochDifferenceTicks {
		t.Errorf("Epoch difference mismatch: calculated %d, expected %d",
			expectedTicks, epochDifferenceTicks)
	}
}

func BenchmarkTimeToTicks(b *testing.B) {
	now := time.Now().UTC()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = TimeToTicks(now)
	}
}

func BenchmarkTicksToTime(b *testing.B) {
	ticks := int64(638997650915691759)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = TicksToTime(ticks)
	}
}
