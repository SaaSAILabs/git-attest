package privacy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedact_AWSAccessKey(t *testing.T) {
	r := NewRedactor()
	input := "use key AKIAIOSFODNN7EXAMPLE to authenticate"
	result := r.Redact(input)
	if strings.Contains(result, "AKIAIOSFODNN7EXAMPLE") {
		t.Errorf("AWS key was not redacted: %s", result)
	}
	if !strings.Contains(result, "[REDACTED]") {
		t.Errorf("expected [REDACTED] placeholder in: %s", result)
	}
}

func TestRedact_AWSSecretKey(t *testing.T) {
	r := NewRedactor()
	input := "aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	result := r.Redact(input)
	if strings.Contains(result, "wJalrXUtnFEMI") {
		t.Errorf("AWS secret was not redacted: %s", result)
	}
}

func TestRedact_JWT(t *testing.T) {
	r := NewRedactor()
	jwt := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
	input := "Authorization: Bearer " + jwt
	result := r.Redact(input)
	if strings.Contains(result, "eyJhbGciOiJ") {
		t.Errorf("JWT was not redacted: %s", result)
	}
}

func TestRedact_IPv4(t *testing.T) {
	r := NewRedactor()
	input := "connect to database at 192.168.1.42 on port 5432"
	result := r.Redact(input)
	if strings.Contains(result, "192.168.1.42") {
		t.Errorf("IPv4 address was not redacted: %s", result)
	}
}

func TestRedact_APIKey(t *testing.T) {
	r := NewRedactor()
	input := "api_key=sk-abc123secret456"
	result := r.Redact(input)
	if strings.Contains(result, "sk-abc123secret456") {
		t.Errorf("API key was not redacted: %s", result)
	}
}

func TestRedact_CleanTextUnchanged(t *testing.T) {
	r := NewRedactor()
	input := "Refactor the login handler to use middleware"
	result := r.Redact(input)
	if result != input {
		t.Errorf("clean text was modified: %q -> %q", input, result)
	}
}

func TestRedact_MultipleSecrets(t *testing.T) {
	r := NewRedactor()
	input := "key AKIAIOSFODNN7EXAMPLE at 10.0.0.1 with api_key=hunter2"
	result := r.Redact(input)
	if strings.Contains(result, "AKIAIOSFODNN7EXAMPLE") {
		t.Error("AWS key survived redaction")
	}
	if strings.Contains(result, "10.0.0.1") {
		t.Error("IP survived redaction")
	}
	if strings.Contains(result, "hunter2") {
		t.Error("API key value survived redaction")
	}
}

func TestLoadTraceFilter_CustomPattern(t *testing.T) {
	dir := t.TempDir()
	filterPath := filepath.Join(dir, ".tracefilter")
	// Custom pattern: redact anything matching "INTERNAL-\d+"
	content := "# Custom company patterns\nINTERNAL-\\d+\n"
	if err := os.WriteFile(filterPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write .tracefilter: %v", err)
	}

	r := NewRedactor()
	if err := r.LoadTraceFilter(filterPath); err != nil {
		t.Fatalf("LoadTraceFilter error: %v", err)
	}

	input := "see ticket INTERNAL-4829 for details"
	result := r.Redact(input)
	if strings.Contains(result, "INTERNAL-4829") {
		t.Errorf("custom pattern was not applied: %s", result)
	}
}

func TestLoadTraceFilter_MissingFile(t *testing.T) {
	r := NewRedactor()
	err := r.LoadTraceFilter("/nonexistent/.tracefilter")
	if err != nil {
		t.Errorf("missing .tracefilter should return nil, got: %v", err)
	}
}

func TestLoadTraceFilter_BlankAndCommentLines(t *testing.T) {
	dir := t.TempDir()
	filterPath := filepath.Join(dir, ".tracefilter")
	content := "# comment line\n\n   \n# another comment\nSECRET_WORD\n"
	if err := os.WriteFile(filterPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write .tracefilter: %v", err)
	}

	r := NewRedactor()
	if err := r.LoadTraceFilter(filterPath); err != nil {
		t.Fatalf("LoadTraceFilter error: %v", err)
	}

	// Only 1 custom pattern should have been added (default count + 1).
	input := "the SECRET_WORD is hidden"
	result := r.Redact(input)
	if strings.Contains(result, "SECRET_WORD") {
		t.Errorf("custom pattern from .tracefilter was not applied: %s", result)
	}
}
