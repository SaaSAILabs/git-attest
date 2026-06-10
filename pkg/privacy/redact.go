package privacy

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

const redactedPlaceholder = "[REDACTED]"

// defaultPatterns covers common secret formats.
var defaultPatterns = []string{
	// AWS Access Key IDs (AKIA...)
	`AKIA[0-9A-Z]{16}`,
	// AWS Secret Access Keys (40-char base64)
	`(?i)aws_secret_access_key\s*[=:]\s*\S+`,
	// JWTs (three base64url segments separated by dots)
	`eyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`,
	// IPv4 addresses (private and public)
	`\b(?:\d{1,3}\.){3}\d{1,3}\b`,
	// Generic API key assignments
	`(?i)(?:api[_-]?key|api[_-]?secret|access[_-]?token)\s*[=:]\s*\S+`,
}

// Redactor holds compiled regex patterns for sanitizing prompt text.
type Redactor struct {
	patterns []*regexp.Regexp
}

// NewRedactor creates a Redactor with the built-in default patterns.
func NewRedactor() *Redactor {
	r := &Redactor{}
	for _, p := range defaultPatterns {
		if re, err := regexp.Compile(p); err == nil {
			r.patterns = append(r.patterns, re)
		}
	}
	return r
}

// LoadTraceFilter reads a .tracefilter file and appends each non-empty,
// non-comment line as an additional regex pattern. Invalid regexes are
// silently skipped. Returns nil if the file does not exist.
func (r *Redactor) LoadTraceFilter(path string) error {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil // no custom filters, not an error
	}
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if re, err := regexp.Compile(line); err == nil {
			r.patterns = append(r.patterns, re)
		}
	}
	return scanner.Err()
}

// Redact replaces all matches of every loaded pattern with [REDACTED].
func (r *Redactor) Redact(input string) string {
	result := input
	for _, re := range r.patterns {
		result = re.ReplaceAllString(result, redactedPlaceholder)
	}
	return result
}
