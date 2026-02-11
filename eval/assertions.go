package eval

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type CheckResult struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail,omitempty"`
}

func runAssertions(output string, assertions []Assertion) []CheckResult {
	results := make([]CheckResult, 0, len(assertions))
	for i, a := range assertions {
		name := strings.TrimSpace(a.Type)
		if name == "" {
			name = fmt.Sprintf("assertion_%d", i+1)
		}
		results = append(results, evaluateAssertion(output, a, name))
	}
	return results
}

func evaluateAssertion(output string, a Assertion, name string) CheckResult {
	t := strings.ToLower(strings.TrimSpace(a.Type))
	switch t {
	case "contains":
		needle := a.Value
		if !a.CaseSensitive {
			if strings.Contains(strings.ToLower(output), strings.ToLower(needle)) {
				return CheckResult{Name: name, Pass: true}
			}
			return CheckResult{Name: name, Pass: false, Detail: fmt.Sprintf("missing substring %q", needle)}
		}
		if strings.Contains(output, needle) {
			return CheckResult{Name: name, Pass: true}
		}
		return CheckResult{Name: name, Pass: false, Detail: fmt.Sprintf("missing substring %q", needle)}

	case "regex":
		r, err := regexp.Compile(a.Pattern)
		if err != nil {
			return CheckResult{Name: name, Pass: false, Detail: fmt.Sprintf("invalid regex: %v", err)}
		}
		if r.MatchString(output) {
			return CheckResult{Name: name, Pass: true}
		}
		return CheckResult{Name: name, Pass: false, Detail: fmt.Sprintf("regex did not match %q", a.Pattern)}

	case "equals":
		if output == a.Value {
			return CheckResult{Name: name, Pass: true}
		}
		return CheckResult{Name: name, Pass: false, Detail: "output mismatch"}

	case "json_valid":
		if json.Valid([]byte(output)) {
			return CheckResult{Name: name, Pass: true}
		}
		return CheckResult{Name: name, Pass: false, Detail: "output is not valid JSON"}

	case "json_schema":
		var value any
		if err := json.Unmarshal([]byte(output), &value); err != nil {
			return CheckResult{Name: name, Pass: false, Detail: fmt.Sprintf("invalid JSON: %v", err)}
		}
		if errs := validateSchema(value, a.Schema, "$", nil); len(errs) > 0 {
			return CheckResult{Name: name, Pass: false, Detail: strings.Join(errs, "; ")}
		}
		return CheckResult{Name: name, Pass: true}

	default:
		return CheckResult{Name: name, Pass: false, Detail: fmt.Sprintf("unknown assertion type %q", a.Type)}
	}
}

func validateSchema(value any, schema map[string]any, path string, errs []string) []string {
	if len(schema) == 0 {
		return errs
	}

	if typ, ok := schema["type"].(string); ok {
		if !matchesType(value, typ) {
			return append(errs, fmt.Sprintf("%s: expected %s", path, typ))
		}
	}

	if enumValues, ok := schema["enum"].([]any); ok {
		found := false
		for _, ev := range enumValues {
			if valuesEqual(value, ev) {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, fmt.Sprintf("%s: value not in enum", path))
		}
	}

	obj, isObj := value.(map[string]any)
	if required, ok := schema["required"].([]any); ok {
		if !isObj {
			errs = append(errs, fmt.Sprintf("%s: required fields expect object", path))
		} else {
			for _, item := range required {
				k, ok := item.(string)
				if !ok || strings.TrimSpace(k) == "" {
					continue
				}
				if _, exists := obj[k]; !exists {
					errs = append(errs, fmt.Sprintf("%s.%s: required field missing", path, k))
				}
			}
		}
	}

	if props, ok := schema["properties"].(map[string]any); ok {
		if !isObj {
			errs = append(errs, fmt.Sprintf("%s: properties expect object", path))
		} else {
			for key, raw := range props {
				subSchema, ok := raw.(map[string]any)
				if !ok {
					continue
				}
				v, exists := obj[key]
				if !exists {
					continue
				}
				errs = validateSchema(v, subSchema, path+"."+key, errs)
			}
		}
	}

	if itemSchemaRaw, ok := schema["items"].(map[string]any); ok {
		arr, ok := value.([]any)
		if !ok {
			errs = append(errs, fmt.Sprintf("%s: items expect array", path))
		} else {
			for i, item := range arr {
				errs = validateSchema(item, itemSchemaRaw, fmt.Sprintf("%s[%d]", path, i), errs)
			}
		}
	}

	return errs
}

func matchesType(value any, typ string) bool {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		switch value.(type) {
		case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return true
		default:
			return false
		}
	case "integer":
		n, ok := value.(float64)
		if !ok {
			return false
		}
		return n == float64(int64(n))
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "null":
		return value == nil
	default:
		return true
	}
}

func valuesEqual(a, b any) bool {
	left, err := json.Marshal(a)
	if err != nil {
		return false
	}
	right, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(left) == string(right)
}
