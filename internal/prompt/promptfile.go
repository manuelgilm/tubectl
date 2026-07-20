package prompt

import (
	"os"
	"gopkg.in/yaml.v3"
	"strings"
	"fmt"
)

type PromptFile struct {
	Template	string `yaml:"template"`
	Vars		[]string 	`yaml:"vars"`
}

func (p *PromptFile) Render(vars map[string]string) (string, error) {
	result := p.Template
	for _, v := range p.Vars {
		val, ok := vars[v]
		if !ok {
			return "", fmt.Errorf("missing variable: %s", v)
		}
		result = strings.ReplaceAll(result, "{"+v+"}", val)
	}
	return result, nil
}

func LoadPromptFile(path string) (*PromptFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading prompt file: %w", err)
	}
	var p PromptFile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing prompt file %s: %w", path, err)
	}
	if p.Template == "" {
		return nil, fmt.Errorf("prompt file %s has empty template", path)
	}
	return &p, nil
}