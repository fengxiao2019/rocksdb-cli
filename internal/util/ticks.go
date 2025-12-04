package util

import (
	"fmt"
	"time"
)

const (
	// .NET Ticks are 100-nanosecond intervals since 0001-01-01 00:00:00 UTC
	ticksPerSecond     = 10000000    // 10^7 ticks per second
	ticksPerNanosecond = 1.0 / 100.0 // 1 tick = 100 nanoseconds
	nanosecondsPerTick = 100         // 100 nanoseconds per tick

	// Difference between .NET epoch (0001-01-01) and Unix epoch (1970-01-01)
	// This is 719162 days
	epochDifferenceTicks = 621355968000000000
)

// TimeToTicks converts a Go time.Time to .NET ticks
// Returns the number of 100-nanosecond intervals since 0001-01-01 00:00:00 UTC
func TimeToTicks(t time.Time) int64 {
	// Get Unix time in nanoseconds
	unixNano := t.UnixNano()

	// Convert nanoseconds to ticks (divide by 100)
	ticks := unixNano / nanosecondsPerTick

	// Add the epoch difference
	return ticks + epochDifferenceTicks
}

// TicksToTime converts .NET ticks to a Go time.Time
// Takes the number of 100-nanosecond intervals since 0001-01-01 00:00:00 UTC
func TicksToTime(ticks int64) time.Time {
	// Subtract the epoch difference to get Unix ticks
	unixTicks := ticks - epochDifferenceTicks

	// Convert ticks to nanoseconds (multiply by 100)
	unixNano := unixTicks * nanosecondsPerTick

	// Create time from Unix nanoseconds
	return time.Unix(0, unixNano).UTC()
}

// FormatTicks formats a ticks value as a readable string
func FormatTicks(ticks int64) string {
	return fmt.Sprintf("%d", ticks)
}

// ParseTicksString parses a ticks string to int64
func ParseTicksString(ticksStr string) (int64, error) {
	var ticks int64
	_, err := fmt.Sscanf(ticksStr, "%d", &ticks)
	return ticks, err
}

// Example usage and verification
func ExampleTicksConversion() {
	// Test 1: Current time
	now := time.Now().UTC()
	ticks := TimeToTicks(now)
	converted := TicksToTime(ticks)

	fmt.Printf("Original:  %s\n", now.Format(time.RFC3339Nano))
	fmt.Printf("Ticks:     %d\n", ticks)
	fmt.Printf("Converted: %s\n", converted.Format(time.RFC3339Nano))
	fmt.Printf("Match: %v\n\n", now.Equal(converted))

	// Test 2: User's example - 2025-11-26T14:44:51.569Z
	testTime, _ := time.Parse(time.RFC3339Nano, "2025-11-26T14:44:51.569Z")
	testTicks := TimeToTicks(testTime)
	fmt.Printf("Test time: %s\n", testTime.Format(time.RFC3339Nano))
	fmt.Printf("Ticks:     %d\n", testTicks)

	// Note: The JavaScript version gives 638997650915690000
	// This is because JS only has millisecond precision
	// Go will give a different result if the original time had sub-millisecond precision

	// Test 3: Reverse conversion with expected value
	expectedTicks := int64(638997650915691759)
	reversedTime := TicksToTime(expectedTicks)
	fmt.Printf("\nExpected ticks: %d\n", expectedTicks)
	fmt.Printf("Reversed time:  %s\n", reversedTime.Format(time.RFC3339Nano))

	// Test 4: Unix epoch
	epoch := time.Unix(0, 0).UTC()
	epochTicks := TimeToTicks(epoch)
	fmt.Printf("\nUnix Epoch: %s\n", epoch.Format(time.RFC3339Nano))
	fmt.Printf("Ticks:      %d\n", epochTicks)
	fmt.Printf("Expected:   %d\n", epochDifferenceTicks)
	fmt.Printf("Match: %v\n", epochTicks == epochDifferenceTicks)
}
