package dsh

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func TestDSHHealth_MissingModelKeyIsNotRobotCredentialsAndErrorsAreContentFree(t *testing.T) {
	t.Setenv("DSH_CC_CONNECT_HEALTH", "1")
	previous := slog.Default()
	var buffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buffer, nil)))
	defer slog.SetDefault(previous)
	reportDSHResult(fmt.Errorf(`json-rpc -32603: no API key for provider route "fake-route"; fake-secret`))
	reportDSHResult(fmt.Errorf("fake-secret ACP unavailable"))
	reportDSHResult(nil)
	output := buffer.String()
	for _, expected := range []string{`\"errorCode\":\"model-key-missing\"`, `\"errorCode\":\"acp\"`, `\"state\":\"available\"`} {
		if !strings.Contains(output, expected) {
			t.Fatalf("missing %s: %s", expected, output)
		}
	}
	for _, private := range []string{"fake-secret", "fake-route", "credentials"} {
		if strings.Contains(output, private) {
			t.Fatal("private error or wrong credential category leaked")
		}
	}
}
