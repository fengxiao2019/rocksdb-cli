package graphchain

import (
	"log"
	"regexp"
	"strings"
	"time"
)

// SecurityManager handles security policies and validation
type SecurityManager struct {
	config            *SecurityConfig
	dangerousPatterns []*regexp.Regexp
	allowedOperations map[string]bool
	queryComplexity   int
	lastResetTime     time.Time
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(config *SecurityConfig) *SecurityManager {
	sm := &SecurityManager{
		config:            config,
		allowedOperations: make(map[string]bool),
		queryComplexity:   0,
		lastResetTime:     time.Now(),
	}

	// Initialize allowed operations map
	for _, op := range config.AllowedOperations {
		sm.allowedOperations[strings.ToLower(op)] = true
	}

	// Initialize dangerous patterns
	sm.initializeDangerousPatterns()

	return sm
}

// initializeDangerousPatterns sets up patterns that are considered dangerous
func (sm *SecurityManager) initializeDangerousPatterns() {
	dangerousPatterns := []string{
		// SQL injection-like patterns
		`(?i)(drop|delete|truncate)\s+(table|database|column)`,
		`(?i)(insert|update)\s+.*\s+(values|set)`,
		`(?i)exec(ute)?\s*\(`,
		`(?i)script\s*:`,

		// System command patterns
		`(?i)(rm\s+-rf|del\s+/|format\s+)`,
		`(?i)(shutdown|reboot|halt)`,
		`(?i)cmd\.exe|bash|sh\s+`,

		// Suspicious data access patterns
		`(?i)(password|secret|token|key)\s*=`,
		`(?i)\.\.\/|\.\.\\`,

		// Large bulk operations
		`(?i)(scan|prefix)\s+.*\s+(limit\s+[1-9]\d{4,}|no\s+limit)`,
	}

	sm.dangerousPatterns = make([]*regexp.Regexp, 0, len(dangerousPatterns))
	for _, pattern := range dangerousPatterns {
		if regex, err := regexp.Compile(pattern); err == nil {
			sm.dangerousPatterns = append(sm.dangerousPatterns, regex)
		} else {
			log.Printf("Warning: Invalid security pattern: %s, error: %v", pattern, err)
		}
	}
}

// ValidateQuery validates a query against security policies
func (sm *SecurityManager) ValidateQuery(query string) bool {
	// Check if in read-only mode
	if sm.config.ReadOnlyMode && sm.containsWriteOperation(query) {
		log.Printf("Security: Write operation blocked in read-only mode: %s", query)
		return false
	}

	// Check for dangerous patterns
	if sm.containsDangerousPatterns(query) {
		log.Printf("Security: Dangerous pattern detected in query: %s", query)
		return false
	}

	// Check query complexity
	if !sm.validateQueryComplexity(query) {
		log.Printf("Security: Query complexity limit exceeded: %s", query)
		return false
	}

	// Check allowed operations
	if !sm.validateAllowedOperations(query) {
		log.Printf("Security: Operation not allowed: %s", query)
		return false
	}

	return true
}

// containsWriteOperation checks if the query contains write operations
func (sm *SecurityManager) containsWriteOperation(query string) bool {
	writePatterns := []string{
		"put", "set", "insert", "update", "delete", "drop", "create",
		"modify", "alter", "write", "save", "store",
	}

	lowerQuery := strings.ToLower(query)
	for _, pattern := range writePatterns {
		if strings.Contains(lowerQuery, pattern) {
			return true
		}
	}

	return false
}

// containsDangerousPatterns checks if the query contains dangerous patterns
func (sm *SecurityManager) containsDangerousPatterns(query string) bool {
	for _, pattern := range sm.dangerousPatterns {
		if pattern.MatchString(query) {
			return true
		}
	}
	return false
}

// validateQueryComplexity validates the complexity of the query
func (sm *SecurityManager) validateQueryComplexity(query string) bool {
	// Reset complexity counter if enough time has passed
	if time.Since(sm.lastResetTime) > time.Minute {
		sm.queryComplexity = 0
		sm.lastResetTime = time.Now()
	}

	// Calculate query complexity
	complexity := sm.calculateQueryComplexity(query)

	// Check if adding this query would exceed the limit
	if sm.queryComplexity+complexity > sm.config.MaxQueryComplexity {
		return false
	}

	// Add to current complexity
	sm.queryComplexity += complexity

	return true
}

// calculateQueryComplexity calculates the complexity score of a query
func (sm *SecurityManager) calculateQueryComplexity(query string) int {
	complexity := 1 // Base complexity

	lowerQuery := strings.ToLower(query)

	// Increase complexity for expensive operations
	if strings.Contains(lowerQuery, "scan") {
		complexity += 3
	}
	if strings.Contains(lowerQuery, "search") {
		complexity += 2
	}
	if strings.Contains(lowerQuery, "prefix") {
		complexity += 2
	}
	if strings.Contains(lowerQuery, "json") {
		complexity += 2
	}
	if strings.Contains(lowerQuery, "export") {
		complexity += 4
	}
	if strings.Contains(lowerQuery, "stats") {
		complexity += 2
	}

	// Increase complexity for large limits
	limitPattern := regexp.MustCompile(`limit\s+(\d+)`)
	if matches := limitPattern.FindStringSubmatch(lowerQuery); len(matches) > 1 {
		if limit := parseInt(matches[1]); limit > 1000 {
			complexity += 3
		} else if limit > 100 {
			complexity += 1
		}
	}

	return complexity
}

// validateAllowedOperations checks if the query uses only allowed operations
func (sm *SecurityManager) validateAllowedOperations(query string) bool {
	lowerQuery := strings.ToLower(query)

	// Extract potential operations from the query
	operations := []string{
		"get", "put", "scan", "prefix", "search", "jsonquery",
		"list", "stats", "export", "create", "drop", "last",
	}

	for _, op := range operations {
		if strings.Contains(lowerQuery, op) {
			if !sm.allowedOperations[op] {
				return false
			}
		}
	}

	return true
}

// IsOperationAllowed checks if a specific operation is allowed
func (sm *SecurityManager) IsOperationAllowed(operation string) bool {
	return sm.allowedOperations[strings.ToLower(operation)]
}

// GetSecurityReport returns a security report
func (sm *SecurityManager) GetSecurityReport() map[string]interface{} {
	return map[string]interface{}{
		"read_only_mode":       sm.config.ReadOnlyMode,
		"audit_enabled":        sm.config.EnableAudit,
		"max_query_complexity": sm.config.MaxQueryComplexity,
		"current_complexity":   sm.queryComplexity,
		"allowed_operations":   sm.config.AllowedOperations,
		"dangerous_patterns":   len(sm.dangerousPatterns),
		"last_reset_time":      sm.lastResetTime,
	}
}

// parseInt safely parses an integer string
func parseInt(s string) int {
	if len(s) == 0 {
		return 0
	}

	result := 0
	for _, char := range s {
		if char >= '0' && char <= '9' {
			result = result*10 + int(char-'0')
		} else {
			break
		}
	}

	return result
}
