package transform

import (
	"fmt"
)

// transformKey applies transformation to key
func (p *transformProcessor) transformKey(key, value string, opts TransformOptions) (string, error) {
	// If no key expression, return original key
	if opts.KeyExpression == "" {
		return key, nil
	}

	// Build context
	context := map[string]interface{}{
		"key":   key,
		"value": value,
	}

	// Execute key transformation
	result, err := p.executor.ExecuteExpression(opts.KeyExpression, context)
	if err != nil {
		return "", fmt.Errorf("key transformation failed: %w", err)
	}

	// Convert result to string
	resultStr, ok := result.(string)
	if !ok {
		resultStr = fmt.Sprintf("%v", result)
	}

	// Handle None/null results - keep original key
	if resultStr == "None" || resultStr == "null" || resultStr == "" {
		return key, nil
	}

	return resultStr, nil
}
