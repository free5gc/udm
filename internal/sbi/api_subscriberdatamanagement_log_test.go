package sbi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/free5gc/udm/internal/logger"
	"github.com/free5gc/udm/pkg/factory"
	logger_util "github.com/free5gc/util/logger"
)

func TestTwoLayerPathHandlerLogForging(t *testing.T) {
	const marker = "FORGED_LOG_LINE_1232"

	// logger.Log is a package global: restore it, and do not use t.Parallel.
	originalOutput := logger.Log.Out
	originalLevel := logger.Log.GetLevel()
	t.Cleanup(func() {
		logger.Log.SetOutput(originalOutput)
		logger.Log.SetLevel(originalLevel)
	})
	logger.Log.SetLevel(logrus.InfoLevel)

	for _, tt := range []struct {
		name       string
		method     string
		injected   string
		wantEscape string
		inOp       bool
	}{
		{"GET SUPI LF", http.MethodGet, "%0A", `\n`, false},
		{"GET SUPI CRLF", http.MethodGet, "%0D%0A", `\r\n`, false},
		{"POST SUPI LF", http.MethodPost, "%0A", `\n`, false},
		{"POST SUPI CRLF", http.MethodPost, "%0D%0A", `\r\n`, false},
		{"GET operation LF", http.MethodGet, "%0A", `\n`, true},
		{"GET operation CRLF", http.MethodGet, "%0D%0A", `\r\n`, true},
		{"GET SUPI U+2028", http.MethodGet, "%E2%80%A8", `\u2028`, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			logger.Log.SetOutput(&logs)

			// Exercise the real Gin middleware and the dispatch handler,
			// without a running NF or an authentication middleware.
			router := logger_util.NewGinWithLogrus(logger.GinLog)
			prefix := factory.UdmSdmResUriPrefix
			router.Any(prefix+"/:supi/:subscriptionId", (&Server{}).TwoLayerPathHandlerFunc)
			url := prefix + "/invalid" + tt.injected + marker + "/unsupported-operation"
			if tt.inOp {
				url = prefix + "/invalid/unsupported-operation" + tt.injected + marker
			}

			req := httptest.NewRequestWithContext(context.Background(), tt.method, url, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			if resp.Code != http.StatusNotFound {
				t.Fatalf("HTTP status = %d, want 404", resp.Code)
			}

			consumerCount, ginCount := 0, 0
			for _, line := range strings.Split(logs.String(), "\n") {
				if !strings.Contains(line, marker) {
					continue
				}
				if !strings.Contains(line, tt.wantEscape) || strings.ContainsAny(line, "\r\u2028\u2029") {
					t.Errorf("unsafe log line: %q", line)
				}
				switch {
				case strings.Contains(line, "TwoLayerPathHandlerFunc"):
					consumerCount++
				case strings.Contains(line, "| 404 |"):
					ginCount++
				default:
					t.Errorf("unexpected forged log entry: %q", line)
				}
			}
			if consumerCount != 1 || ginCount != 1 {
				t.Errorf("want one sanitized event per logger, got Consumer=%d, Gin=%d; logs:\n%s",
					consumerCount, ginCount, logs.String())
			}
			if strings.Count(logs.String(), marker) != 2 {
				t.Errorf("marker must appear exactly twice, got:\n%s", logs.String())
			}
		})
	}
}
