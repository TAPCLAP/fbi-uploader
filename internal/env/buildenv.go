package env

import (
	"fmt"
	"os"
	"strings"
)

// ApplyBuildEnv reads BUILD_ENV_PATH if set and applies KEY=VALUE lines to the process
// environment. Variables already set in the environment are not overwritten.
func ApplyBuildEnv() error {
	path := strings.TrimSpace(os.Getenv("BUILD_ENV_PATH"))
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read BUILD_ENV_PATH: %w", err)
	}

	for lineNum, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if after, ok := strings.CutPrefix(line, "export "); ok {
			line = strings.TrimSpace(after)
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("BUILD_ENV_PATH line %d: expected KEY=VALUE", lineNum+1)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("BUILD_ENV_PATH line %d: empty variable name", lineNum+1)
		}

		value = unquoteEnvValue(strings.TrimSpace(value))
		if os.Getenv(key) != "" {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s from BUILD_ENV_PATH: %w", key, err)
		}
	}
	return nil
}

func unquoteEnvValue(value string) string {
	if len(value) < 2 {
		return value
	}
	switch value[0] {
	case '"':
		if value[len(value)-1] == '"' {
			return value[1 : len(value)-1]
		}
	case '\'':
		if value[len(value)-1] == '\'' {
			return value[1 : len(value)-1]
		}
	}
	return value
}
