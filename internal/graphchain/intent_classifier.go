// intent_classifier.go
package graphchain

import (
	"regexp"
	"strings"
)

type IntentClassifier struct {
	patterns map[string]*regexp.Regexp
}

func NewIntentClassifier() *IntentClassifier {
	patterns := map[string]*regexp.Regexp{
		"get_value":  regexp.MustCompile(`(?i)\b(get|fetch|retrieve|show|find)\s+.*\b(key|value|data)\b`),
		"scan_keys":  regexp.MustCompile(`(?i)\b(list|show|scan|all)\s+.*\b(keys?|entries)\b`),
		"store_data": regexp.MustCompile(`(?i)\b(put|set|store|save|insert|add)\b`),
		"list_cf":    regexp.MustCompile(`(?i)\b(list|show)\s+.*\b(column\s+famil|cf)\b`),
		"query_json": regexp.MustCompile(`(?i)\bjson\b.*\b(query|search|find)\b`),
		"get_stats":  regexp.MustCompile(`(?i)\b(stats?|statistics|info|status)\b`),
	}

	return &IntentClassifier{patterns: patterns}
}

func (ic *IntentClassifier) ClassifyIntent(query string) string {
	query = strings.ToLower(strings.TrimSpace(query))

	for intent, pattern := range ic.patterns {
		if pattern.MatchString(query) {
			return intent
		}
	}

	return "general_query"
}
