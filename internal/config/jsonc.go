package config

import "encoding/json"

// parseJSONC parses JSON content with optional C-style comments (// and /* */).
// It first attempts standard json.Unmarshal; if that fails, it strips comments
// and retries. This ensures valid JSON always works without preprocessing.
func parseJSONC(data []byte, v interface{}) error {
	if err := json.Unmarshal(data, v); err == nil {
		return nil
	}
	stripped := stripJSONComments(data)
	return json.Unmarshal(stripped, v)
}

// stripJSONComments removes // line comments and /* */ block comments from JSON
// content while preserving string literals and escaped characters.
func stripJSONComments(data []byte) []byte {
	result := make([]byte, 0, len(data))
	i := 0
	for i < len(data) {
		switch data[i] {
		case '"':
			result = append(result, data[i])
			i++
			for i < len(data) && data[i] != '"' {
				if data[i] == '\\' && i+1 < len(data) {
					result = append(result, data[i], data[i+1])
					i += 2
				} else {
					result = append(result, data[i])
					i++
				}
			}
			if i < len(data) {
				result = append(result, data[i]) // closing quote
				i++
			}
		case '/':
			if i+1 < len(data) {
				if data[i+1] == '/' {
					// Skip line comment until newline
					i += 2
					for i < len(data) && data[i] != '\n' {
						i++
					}
				} else if data[i+1] == '*' {
					// Skip block comment until */
					i += 2
					for i+1 < len(data) && !(data[i] == '*' && data[i+1] == '/') {
						i++
					}
					if i+1 < len(data) {
						i += 2 // skip */
					}
				} else {
					result = append(result, data[i])
					i++
				}
			} else {
				result = append(result, data[i])
				i++
			}
		default:
			result = append(result, data[i])
			i++
		}
	}
	return result
}
