package graphchain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSecurityManager(t *testing.T) {
	config := &SecurityConfig{
		EnableAudit:        true,
		ReadOnlyMode:       false,
		MaxQueryComplexity: 10,
		AllowedOperations:  []string{"get", "scan", "prefix"},
	}

	sm := NewSecurityManager(config)

	assert.NotNil(t, sm)
	assert.Equal(t, config, sm.config)
	assert.NotEmpty(t, sm.allowedOperations)
	assert.True(t, sm.allowedOperations["get"])
	assert.True(t, sm.allowedOperations["scan"])
	assert.True(t, sm.allowedOperations["prefix"])
	assert.False(t, sm.allowedOperations["put"])
	assert.NotEmpty(t, sm.dangerousPatterns)
	assert.Equal(t, 0, sm.queryComplexity)
}

func TestSecurityManager_ValidateQuery_ReadOnlyMode(t *testing.T) {
	config := &SecurityConfig{
		ReadOnlyMode:       true,
		MaxQueryComplexity: 10,
		AllowedOperations:  []string{"get", "scan", "prefix", "put", "search"},
	}

	sm := NewSecurityManager(config)

	testCases := []struct {
		name     string
		query    string
		expected bool
	}{
		{"read operation allowed", "get user:1", true},
		{"scan operation allowed", "scan range", true},
		{"prefix operation allowed", "prefix search", true},
		{"put operation blocked", "put user:1 value", false},
		{"update operation blocked", "update user data", false},
		{"delete operation blocked", "delete user:1", false},
		{"create operation blocked", "create new record", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sm.ValidateQuery(tc.query)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSecurityManager_ValidateQuery_DangerousPatterns(t *testing.T) {
	config := &SecurityConfig{
		ReadOnlyMode:       false,
		MaxQueryComplexity: 100,
		AllowedOperations:  []string{"get", "scan", "prefix", "put", "drop", "delete"},
	}

	sm := NewSecurityManager(config)

	testCases := []struct {
		name     string
		query    string
		expected bool
	}{
		{"safe query", "get user:1", true},
		{"sql injection attempt", "DROP TABLE users", false},
		{"delete table attempt", "DELETE TABLE sensitive", false},
		{"script injection", "script: alert('xss')", false},
		{"system command", "rm -rf /", false},
		{"password access", "password = secret123", false},
		{"directory traversal", "../../../etc/passwd", false},
		{"bulk scan with huge limit", "scan range limit 99999", false},
		{"normal scan with reasonable limit", "scan range limit 100", true},
		{"executable command", "cmd.exe /c dir", false},
		{"bash command", "bash -c 'rm -rf'", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sm.ValidateQuery(tc.query)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSecurityManager_ValidateQuery_QueryComplexity(t *testing.T) {
	config := &SecurityConfig{
		ReadOnlyMode:       false,
		MaxQueryComplexity: 5,
		AllowedOperations:  []string{"get", "scan", "export", "stats"},
	}

	sm := NewSecurityManager(config)

	// Test simple queries that should pass
	assert.True(t, sm.ValidateQuery("get user:1"))
	assert.True(t, sm.ValidateQuery("get user:2"))

	// Test complex query that should push us over the limit
	assert.False(t, sm.ValidateQuery("export all data scan everything"))

	// Reset should happen after 1 minute, let's test the complexity accumulation
	sm.queryComplexity = 4
	assert.False(t, sm.ValidateQuery("scan large dataset"))

	// Test complexity reset by manipulating time
	sm.lastResetTime = time.Now().Add(-2 * time.Minute)
	assert.True(t, sm.ValidateQuery("scan data"))
}

func TestSecurityManager_ValidateQuery_AllowedOperations(t *testing.T) {
	config := &SecurityConfig{
		ReadOnlyMode:       false,
		MaxQueryComplexity: 100,
		AllowedOperations:  []string{"get", "scan"},
	}

	sm := NewSecurityManager(config)

	testCases := []struct {
		name     string
		query    string
		expected bool
	}{
		{"allowed get operation", "get user:1", true},
		{"allowed scan operation", "scan range", true},
		{"disallowed put operation", "put user:1 value", false},
		{"disallowed prefix operation", "prefix search", false},
		{"disallowed jsonquery operation", "jsonquery field", false},
		{"disallowed export operation", "export data", false},
		{"query without specific operation", "find something", true}, // No specific operation detected
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sm.ValidateQuery(tc.query)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSecurityManager_CalculateQueryComplexity(t *testing.T) {
	config := &SecurityConfig{
		MaxQueryComplexity: 100,
		AllowedOperations:  []string{"get", "scan", "export", "stats", "jsonquery", "prefix", "search"},
	}

	sm := NewSecurityManager(config)

	testCases := []struct {
		name               string
		query              string
		expectedComplexity int
	}{
		{"simple get", "get user:1", 1},
		{"scan operation", "scan range", 4},                       // 1 base + 3 for scan
		{"search operation", "search data", 3},                    // 1 base + 2 for search
		{"json query", "jsonquery field", 3},                      // 1 base + 2 for json
		{"export operation", "export data", 5},                    // 1 base + 4 for export
		{"stats operation", "stats database", 3},                  // 1 base + 2 for stats
		{"prefix operation", "prefix user:", 3},                   // 1 base + 2 for prefix
		{"multiple operations", "scan and export and search", 10}, // 1 + 3 + 4 + 2
		{"with small limit", "scan limit 50", 4},                  // scan complexity only
		{"with medium limit", "scan limit 500", 5},                // scan + 1 for medium limit
		{"with large limit", "scan limit 5000", 7},                // scan + 3 for large limit
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			complexity := sm.calculateQueryComplexity(tc.query)
			assert.Equal(t, tc.expectedComplexity, complexity)
		})
	}
}

func TestSecurityManager_IsOperationAllowed(t *testing.T) {
	config := &SecurityConfig{
		AllowedOperations: []string{"GET", "Scan", "prefix"},
	}

	sm := NewSecurityManager(config)

	// Test case sensitivity
	assert.True(t, sm.IsOperationAllowed("get"))
	assert.True(t, sm.IsOperationAllowed("GET"))
	assert.True(t, sm.IsOperationAllowed("scan"))
	assert.True(t, sm.IsOperationAllowed("SCAN"))
	assert.True(t, sm.IsOperationAllowed("prefix"))
	assert.True(t, sm.IsOperationAllowed("PREFIX"))

	// Test disallowed operations
	assert.False(t, sm.IsOperationAllowed("put"))
	assert.False(t, sm.IsOperationAllowed("delete"))
	assert.False(t, sm.IsOperationAllowed("export"))
}

func TestSecurityManager_ContainsWriteOperation(t *testing.T) {
	config := &SecurityConfig{}
	sm := NewSecurityManager(config)

	testCases := []struct {
		name     string
		query    string
		expected bool
	}{
		{"get operation", "get user:1", false},
		{"scan operation", "scan range", false},
		{"put operation", "put user:1 value", true},
		{"set operation", "set config", true},
		{"insert operation", "insert new record", true},
		{"update operation", "update user data", true},
		{"delete operation", "delete user:1", true},
		{"drop operation", "drop table", true},
		{"create operation", "create new table", true},
		{"modify operation", "modify settings", true},
		{"alter operation", "alter structure", true},
		{"write operation", "write data", true},
		{"save operation", "save configuration", true},
		{"store operation", "store value", true},
		{"mixed case", "PUT user data", true},
		{"query with update word in context", "get user_updated_time", true}, // "update" is detected as write operation
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sm.containsWriteOperation(tc.query)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSecurityManager_ContainsDangerousPatterns(t *testing.T) {
	config := &SecurityConfig{}
	sm := NewSecurityManager(config)

	testCases := []struct {
		name     string
		query    string
		expected bool
	}{
		{"safe query", "get user data", false},
		{"drop table", "DROP TABLE users", true},
		{"delete table", "DELETE TABLE sensitive", true},
		{"sql injection", "'; DROP TABLE users; --", true},
		{"script tag", "script: alert('test')", true},
		{"system command rm", "rm -rf /home", true},
		{"system command del", "del /f important.txt", true},
		{"executable", "cmd.exe /c dir", true},
		{"bash command", "bash script.sh", true},
		{"password exposure", "password = secret", true},
		{"directory traversal", "../../../etc/passwd", true},
		{"large limit", "scan prefix user: limit 100000", true},
		{"no limit", "scan all no limit", true},
		{"case insensitive", "drop table Users", true},
		{"insert values", "INSERT INTO users VALUES (1, 'test')", true},
		{"update set", "UPDATE users SET name = 'hacker'", true},
		{"execute function", "exec('malicious code')", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sm.containsDangerousPatterns(tc.query)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSecurityManager_GetSecurityReport(t *testing.T) {
	config := &SecurityConfig{
		EnableAudit:        true,
		ReadOnlyMode:       true,
		MaxQueryComplexity: 15,
		AllowedOperations:  []string{"get", "scan"},
	}

	sm := NewSecurityManager(config)
	sm.queryComplexity = 5

	report := sm.GetSecurityReport()

	assert.Equal(t, true, report["read_only_mode"])
	assert.Equal(t, true, report["audit_enabled"])
	assert.Equal(t, 15, report["max_query_complexity"])
	assert.Equal(t, 5, report["current_complexity"])
	assert.Equal(t, []string{"get", "scan"}, report["allowed_operations"])
	assert.Greater(t, report["dangerous_patterns"].(int), 0)
	assert.NotNil(t, report["last_reset_time"])
}

func TestSecurityManager_ParseInt(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected int
	}{
		{"valid number", "123", 123},
		{"number with letters", "123abc", 123},
		{"empty string", "", 0},
		{"letters only", "abc", 0},
		{"zero", "0", 0},
		{"large number", "999999", 999999},
		{"negative sign ignored", "-123", 0}, // parseInt doesn't handle negative
		{"decimal ignored", "123.45", 123},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseInt(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSecurityManager_ComplexityReset(t *testing.T) {
	config := &SecurityConfig{
		MaxQueryComplexity: 10,
		AllowedOperations:  []string{"get", "scan"},
	}

	sm := NewSecurityManager(config)

	// Set high complexity
	sm.queryComplexity = 8

	// Test that we can't exceed limit
	assert.False(t, sm.ValidateQuery("scan large dataset")) // This should add 4, total would be 12 > 10

	// Simulate time passage for reset
	sm.lastResetTime = time.Now().Add(-2 * time.Minute)

	// Now the same query should pass as complexity is reset
	assert.True(t, sm.ValidateQuery("scan dataset"))
	assert.Equal(t, 4, sm.queryComplexity) // Should be reset and then set to query complexity
}

func TestSecurityManager_AllowedOperationsEmpty(t *testing.T) {
	config := &SecurityConfig{
		ReadOnlyMode:       false,
		MaxQueryComplexity: 100,
		AllowedOperations:  []string{}, // Empty list means no operations allowed
	}

	sm := NewSecurityManager(config)

	// No operations should be allowed when list is empty
	testQueries := []string{
		"get user:1",
		"scan range",
		"put user:1 value",
		"prefix search",
		"export data",
	}

	for _, query := range testQueries {
		assert.False(t, sm.ValidateQuery(query), "Query should be blocked when no operations allowed: %s", query)
	}
}

func TestSecurityManager_MultipleSecurityChecks(t *testing.T) {
	config := &SecurityConfig{
		ReadOnlyMode:       true,
		MaxQueryComplexity: 3,
		AllowedOperations:  []string{"get"},
	}

	sm := NewSecurityManager(config)

	// This should fail multiple security checks
	dangerousWriteQuery := "DROP TABLE users AND scan everything"

	assert.False(t, sm.ValidateQuery(dangerousWriteQuery))

	// Reset complexity for isolated test
	sm.queryComplexity = 0
	sm.lastResetTime = time.Now()

	// Test each failure reason individually
	assert.True(t, sm.containsWriteOperation("put user:1 value"))
	assert.True(t, sm.containsDangerousPatterns("DROP TABLE users"))

	// Test complexity limit
	sm.queryComplexity = 0
	assert.False(t, sm.validateQueryComplexity("scan and export and search")) // Should exceed limit
}

func TestSecurityManager_CaseInsensitivity(t *testing.T) {
	config := &SecurityConfig{
		ReadOnlyMode:       false,
		MaxQueryComplexity: 100,
		AllowedOperations:  []string{"GET", "SCAN"},
	}

	sm := NewSecurityManager(config)

	// Test case insensitive operations
	testCases := []struct {
		operation string
		expected  bool
	}{
		{"get", true},
		{"GET", true},
		{"Get", true},
		{"scan", true},
		{"SCAN", true},
		{"Scan", true},
		{"put", false},
		{"PUT", false},
		{"Put", false},
	}

	for _, tc := range testCases {
		t.Run(tc.operation, func(t *testing.T) {
			assert.Equal(t, tc.expected, sm.IsOperationAllowed(tc.operation))
		})
	}
}
