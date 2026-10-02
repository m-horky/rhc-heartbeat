// Package main provides a playground for exporting one heartbeat as OTLP/JSON.
package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
	"github.com/m-horky/rhc-heartbeat/pkg/version"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

const (
	defaultEndpoint       = "http://otel:4318/v1/logs"
	clientCertificatePath = "/etc/pki/consumer/cert.pem"
	clientKeyPath         = "/etc/pki/consumer/cert/key.pem"
	requestTimeout        = 10 * time.Second
	connectTimeout        = 5 * time.Second
	maxResponseBody       = 64 * 1024
)

// main runs the heartbeat export playground and reports failures.
func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

	if err := run(context.Background()); err != nil {
		slog.Error("heartbeat export failed", "err", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run collects a periodic heartbeat and exports it to the configured OTLP/HTTP endpoint.
func run(ctx context.Context) error {
	hb, err := heartbeat.Get(ctx, heartbeat.TriggerPing)
	if err != nil {
		return fmt.Errorf("collect heartbeat: %w", err)
	}

	endpoint := defaultEndpoint

	client, err := newHTTPClient(endpoint)
	if err != nil {
		return err
	}
	defer client.CloseIdleConnections()

	logs := buildLogs(hb, time.Now())
	if err := exportLogs(ctx, client, endpoint, logs); err != nil {
		return err
	}

	slog.Info("heartbeat exported", "endpoint", endpoint)

	return nil
}

// newHTTPClient creates an HTTP client for the endpoint, using mTLS for HTTPS URLs.
func newHTTPClient(endpoint string) (*http.Client, error) {
	parsed, err := url.ParseRequestURI(endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("invalid OTLP endpoint URL %q", endpoint)
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = (&net.Dialer{Timeout: connectTimeout}).DialContext
	transport.IdleConnTimeout = 30 * time.Second

	if parsed.Scheme == "https" {
		certificate, err := tls.LoadX509KeyPair(clientCertificatePath, clientKeyPath)
		if err != nil {
			return nil, fmt.Errorf("load OTLP client certificate: %w", err)
		}

		transport.TLSClientConfig = &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{certificate},
		}
	}

	return &http.Client{Transport: transport, Timeout: requestTimeout}, nil
}

// buildLogs maps one heartbeat to an OTLP log record and its resource attributes.
func buildLogs(hb heartbeat.Heartbeat, observedAt time.Time) plog.Logs {
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	resource := resourceLogs.Resource()
	resource.Attributes().PutStr("service.name", "rhc-heartbeat")
	resource.Attributes().PutStr("service.version", version.Version)

	resource.Attributes().PutStr("host.id", hb.HostID)
	resource.Attributes().PutStr("host.org", hb.HostOrg)
	resource.Attributes().PutStr("host.boot_id", hb.BootID)

	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	scopeLogs.Scope().SetName("github.com/m-horky/rhc-heartbeat")

	record := scopeLogs.LogRecords().AppendEmpty()
	record.SetTimestamp(pcommon.NewTimestampFromTime(hb.TimeUnix))
	record.SetObservedTimestamp(pcommon.NewTimestampFromTime(observedAt))
	record.SetSeverityNumber(plog.SeverityNumberInfo)
	record.SetSeverityText("INFO")
	record.Body().SetStr("Heartbeat collected")

	attributes := record.Attributes()
	attributes.PutStr("heartbeat.trigger", string(hb.Trigger))
	attributes.PutInt("heartbeat.time_monotonic", hb.TimeMonotonic.Nanoseconds())
	attributes.PutStr("heartbeat.time_quality", string(hb.TimeQuality))

	return logs
}

// exportLogs sends OTLP/JSON logs and reports HTTP or partial-success failures.
func exportLogs(ctx context.Context, client *http.Client, endpoint string, logs plog.Logs) error {
	payload, err := (&plog.JSONMarshaler{}).MarshalLogs(logs)
	if err != nil {
		return fmt.Errorf("marshal OTLP/JSON logs: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create OTLP request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("send OTLP request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBody))
	if err != nil {
		return fmt.Errorf("read OTLP response: %w", err)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("OTLP endpoint returned %s: %s", response.Status, strings.TrimSpace(string(responseBody)))
	}

	if len(responseBody) == 0 {
		return nil
	}

	var result exportLogsResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return fmt.Errorf("decode OTLP response: %w", err)
	}

	if result.PartialSuccess != nil &&
		(result.PartialSuccess.RejectedLogRecords > 0 || result.PartialSuccess.ErrorMessage != "") {
		return fmt.Errorf("OTLP endpoint partially rejected logs: rejected=%d message=%q",
			result.PartialSuccess.RejectedLogRecords, result.PartialSuccess.ErrorMessage)
	}

	return nil
}

type exportLogsResponse struct {
	PartialSuccess *partialSuccess `json:"partialSuccess"`
}

type partialSuccess struct {
	RejectedLogRecords int64  `json:"rejectedLogRecords,string"`
	ErrorMessage       string `json:"errorMessage"`
}
