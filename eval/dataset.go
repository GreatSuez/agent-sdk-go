package eval

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Case struct {
	ID             string         `json:"id,omitempty"`
	Input          string         `json:"input"`
	ExpectedOutput string         `json:"expectedOutput,omitempty"`
	RequiredTools  []string       `json:"requiredTools,omitempty"`
	ForbiddenTools []string       `json:"forbiddenTools,omitempty"`
	Assertions     []Assertion    `json:"assertions,omitempty"`
	JudgeRubric    string         `json:"judgeRubric,omitempty"`
	MinJudgeScore  float64        `json:"minJudgeScore,omitempty"`
	Tags           []string       `json:"tags,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type Assertion struct {
	Type          string         `json:"type"`
	Value         string         `json:"value,omitempty"`
	Pattern       string         `json:"pattern,omitempty"`
	Schema        map[string]any `json:"schema,omitempty"`
	CaseSensitive bool           `json:"caseSensitive,omitempty"`
}

func LoadJSONL(path string) ([]Case, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open dataset: %w", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	cases := make([]Case, 0, 64)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		var c Case
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			return nil, fmt.Errorf("parse dataset line %d: %w", lineNo, err)
		}
		c.Input = strings.TrimSpace(c.Input)
		if c.Input == "" {
			return nil, fmt.Errorf("dataset line %d: input is required", lineNo)
		}
		if strings.TrimSpace(c.ID) == "" {
			c.ID = fmt.Sprintf("case-%d", len(cases)+1)
		}
		cases = append(cases, c)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan dataset: %w", err)
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("dataset %q has no cases", path)
	}
	return cases, nil
}
