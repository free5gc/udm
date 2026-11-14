package logger

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	logger_util "github.com/free5gc/util/logger"
)

func TestGinLogSanitizer(t *testing.T) {
	tests := []struct {
		name     string
		category string
		input    string
		want     string
	}{
		{"LF", "GIN", "bad\nFORGED", `bad\nFORGED`},
		{"CRLF", "GIN", "bad\r\nFORGED", `bad\r\nFORGED`},
		{"CR", "GIN", "bad\rFORGED", `bad\rFORGED`},
		{"ESC", "GIN", "bad\x1b[2JFORGED", `bad\x1b[2JFORGED`},
		{"line separator", "GIN", "bad\u2028FORGED", `bad\u2028FORGED`},
		{"paragraph separator", "GIN", "bad\u2029FORGED", `bad\u2029FORGED`},
		{"normal path", "GIN", "/nudm-sdm/v2/imsi-001/am-data", "/nudm-sdm/v2/imsi-001/am-data"},
		{"non GIN log left unchanged", "Consumer", "bad\nFORGED", "bad\nFORGED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &logrus.Entry{
				Data:    logrus.Fields{logger_util.FieldCategory: tt.category},
				Message: tt.input,
			}
			if err := (ginLogSanitizer{}).Fire(entry); err != nil {
				t.Fatal(err)
			}
			if entry.Message != tt.want {
				t.Errorf("message = %q, want %q", entry.Message, tt.want)
			}
		})
	}
}

// rawMessageFormatter writes the entry message as-is, so the test sees exactly
// what the hook produced (logrus.TextFormatter would quote and re-escape it).
type rawMessageFormatter struct{}

func (rawMessageFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(entry.Message + "\n"), nil
}

func TestGinLogSanitizerWithLogrus(t *testing.T) {
	log := logrus.New()
	log.SetFormatter(rawMessageFormatter{})
	var output bytes.Buffer
	log.SetOutput(&output)
	log.AddHook(ginLogSanitizer{})
	log.WithField(logger_util.FieldCategory, "GIN").Info("GET /demo\nFORGED")

	// Only the record terminator may be a physical line break.
	if got := strings.Count(output.String(), "\n"); got != 1 {
		t.Fatalf("expected one physical log line, got %d: %q", got, output.String())
	}
	if !strings.Contains(output.String(), `GET /demo\nFORGED`) {
		t.Fatalf("Gin log not escaped: %q", output.String())
	}
}
