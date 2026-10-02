package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
	"github.com/m-horky/rhc-heartbeat/pkg/version"
	"go.opentelemetry.io/collector/pdata/plog"
)

type otlpJSONDocument struct {
	ResourceLogs []struct {
		Resource struct {
			Attributes []otlpJSONAttribute `json:"attributes"`
		} `json:"resource"`
		ScopeLogs []struct {
			Scope struct {
				Name string `json:"name"`
			} `json:"scope"`
			LogRecords []struct {
				TimeUnixNano         string `json:"timeUnixNano"`
				ObservedTimeUnixNano string `json:"observedTimeUnixNano"`
				SeverityText         string `json:"severityText"`
				Body                 struct {
					StringValue string `json:"stringValue"`
				} `json:"body"`
				Attributes []otlpJSONAttribute `json:"attributes"`
			} `json:"logRecords"`
		} `json:"scopeLogs"`
	} `json:"resourceLogs"`
}

type otlpJSONAttribute struct {
	Key   string `json:"key"`
	Value struct {
		StringValue string `json:"stringValue"`
		IntValue    string `json:"intValue"`
	} `json:"value"`
}

type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip invokes the test function as an HTTP transport.
func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

// TestBuildLogsMarshalsHeartbeatAsOTLPJSON verifies heartbeat fields use the OTLP/JSON shape.
//
// Given a heartbeat with identity and time data
// When it is converted to pdata logs and marshaled
// Then its timestamps, resource attributes including service version, and record attributes appear in OTLP/JSON.
func TestBuildLogsMarshalsHeartbeatAsOTLPJSON(t *testing.T) {
	hb := heartbeat.Heartbeat{
		HostID: "host-123", HostOrg: "org-456", BootID: "boot-789",
		TimeMonotonic: 42*time.Second + 123*time.Nanosecond,
		TimeUnix:      time.Unix(1_700_000_000, 123),
		TimeQuality:   "sync:0.001", Trigger: heartbeat.TriggerPing,
	}

	logs := buildLogs(hb, time.Unix(1_700_000_001, 0))

	payload, err := (&plog.JSONMarshaler{}).MarshalLogs(logs)
	if err != nil {
		t.Fatalf("marshal OTLP/JSON: %v", err)
	}

	var document otlpJSONDocument
	if err := json.Unmarshal(payload, &document); err != nil {
		t.Fatalf("decode marshaled OTLP/JSON: %v", err)
	}

	if len(document.ResourceLogs) != 1 {
		t.Fatalf("expected one resourceLogs entry, got %d", len(document.ResourceLogs))
	}

	resourceLogs := document.ResourceLogs[0]
	if len(resourceLogs.ScopeLogs) != 1 || len(resourceLogs.ScopeLogs[0].LogRecords) != 1 {
		t.Fatalf("unexpected OTLP log hierarchy: %s", payload)
	}

	if findAttribute(resourceLogs.Resource.Attributes, "service.version").Value.StringValue != version.Version ||
		findAttribute(resourceLogs.Resource.Attributes, "host.id").Value.StringValue != hb.HostID {
		t.Errorf("service.version or host.id resource attribute was not serialized")
	}

	if findAttribute(resourceLogs.Resource.Attributes, "host.org").Value.StringValue != hb.HostOrg {
		t.Errorf("host.org resource attribute was not serialized")
	}

	scopeLogs := resourceLogs.ScopeLogs[0]
	if scopeLogs.Scope.Name != "github.com/m-horky/rhc-heartbeat" {
		t.Errorf("unexpected scope name %q", scopeLogs.Scope.Name)
	}

	record := scopeLogs.LogRecords[0]
	if record.TimeUnixNano != "1700000000000000123" || record.ObservedTimeUnixNano != "1700000001000000000" {
		t.Errorf("unexpected timestamps: event=%q observed=%q", record.TimeUnixNano, record.ObservedTimeUnixNano)
	}

	if record.SeverityText != "INFO" || record.Body.StringValue != "Heartbeat collected" {
		t.Errorf("unexpected log record: severity=%q body=%q", record.SeverityText, record.Body.StringValue)
	}

	if findAttribute(record.Attributes, "heartbeat.trigger").Value.StringValue != string(hb.Trigger) {
		t.Errorf("heartbeat trigger attribute was not serialized")
	}

	if findAttribute(record.Attributes, "heartbeat.time_monotonic").Value.IntValue != "42000000123" {
		t.Errorf("heartbeat monotonic time attribute was not serialized as an integer")
	}
}

// TestExportLogsPostsOTLPJSON verifies export sends JSON to the requested logs route.
//
// Given a valid OTLP log payload and an HTTP transport
// When the exporter posts the payload
// Then it sends OTLP/JSON and accepts an empty partial-success response.
func TestExportLogsPostsOTLPJSON(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost {
			t.Errorf("unexpected HTTP method %q", request.Method)
		}

		if request.URL.String() != defaultEndpoint {
			t.Errorf("unexpected request URL %q", request.URL)
		}

		if request.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected Content-Type %q", request.Header.Get("Content-Type"))
		}

		body, err := io.ReadAll(request.Body)
		if err != nil {
			return nil, fmt.Errorf("read test request body: %w", err)
		}

		if _, err := (&plog.JSONUnmarshaler{}).UnmarshalLogs(body); err != nil {
			t.Errorf("decode OTLP/JSON request: %v", err)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"partialSuccess":{}}`)),
			Request:    request,
		}, nil
	})}

	logs := buildLogs(heartbeat.Heartbeat{TimeUnix: time.Unix(1_700_000_000, 123)}, time.Unix(1_700_000_001, 0))
	if err := exportLogs(context.Background(), client, defaultEndpoint, logs); err != nil {
		t.Fatalf("export OTLP logs: %v", err)
	}
}

// findAttribute returns the OTLP attribute with the requested key or its zero value.
func findAttribute(attributes []otlpJSONAttribute, key string) otlpJSONAttribute {
	for _, attribute := range attributes {
		if attribute.Key == key {
			return attribute
		}
	}

	return otlpJSONAttribute{}
}
