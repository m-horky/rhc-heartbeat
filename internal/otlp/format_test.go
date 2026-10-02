package otlp

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
	"github.com/m-horky/rhc-heartbeat/pkg/version"
	"go.opentelemetry.io/collector/pdata/plog"
)

type otlpJSONDocument struct {
	ResourceLogs []otlpJSONResourceLogs `json:"resourceLogs"`
}

type otlpJSONResourceLogs struct {
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
}

type otlpJSONAttribute struct {
	Key   string `json:"key"`
	Value struct {
		StringValue string `json:"stringValue"`
		IntValue    string `json:"intValue"`
	} `json:"value"`
}

// TestBuildLogsMarshalsHeartbeatAsOTLPJSON verifies heartbeats retain their OTLP resource and record fields.
//
// Given heartbeats with distinct boot IDs and event data
// When they are converted to pdata logs and marshaled
// Then each heartbeat has its own resource attributes and a correctly timestamped OTLP log record.
func TestBuildLogsMarshalsHeartbeatAsOTLPJSON(t *testing.T) {
	t.Parallel()

	first := heartbeat.Heartbeat{
		HostID: "host-123", HostOrg: "org-456", BootID: "boot-789",
		TimeMonotonic: 42*time.Second + 123*time.Nanosecond,
		TimeUnix:      time.Unix(1_700_000_000, 123),
		TimeQuality:   "sync:0.001", Trigger: heartbeat.TriggerPing,
	}
	second := heartbeat.Heartbeat{
		HostID: "host-123", HostOrg: "org-456", BootID: "boot-new",
		TimeUnix: time.Unix(1_700_000_100, 456), Trigger: heartbeat.TriggerOff,
	}
	observedAt := time.Unix(1_700_000_001, 0)

	payload, err := (&plog.JSONMarshaler{}).MarshalLogs(buildLogs([]heartbeat.Heartbeat{first, second}, observedAt))
	if err != nil {
		t.Fatalf("marshal OTLP/JSON: %v", err)
	}

	var document otlpJSONDocument
	if err := json.Unmarshal(payload, &document); err != nil {
		t.Fatalf("decode marshaled OTLP/JSON: %v", err)
	}

	if len(document.ResourceLogs) != 2 {
		t.Fatalf("expected one resourceLogs entry per heartbeat, got %d", len(document.ResourceLogs))
	}

	assertFormattedHeartbeat(t, document.ResourceLogs[0], first, observedAt)
	assertFormattedHeartbeat(t, document.ResourceLogs[1], second, observedAt)
}

// assertFormattedHeartbeat verifies one OTLP resource and record match the supplied heartbeat.
func assertFormattedHeartbeat(
	t *testing.T,
	resourceLogs otlpJSONResourceLogs,
	hb heartbeat.Heartbeat,
	observedAt time.Time,
) {
	t.Helper()

	if len(resourceLogs.ScopeLogs) != 1 || len(resourceLogs.ScopeLogs[0].LogRecords) != 1 {
		t.Fatalf("unexpected OTLP log hierarchy: %#v", resourceLogs)
	}

	if findAttribute(resourceLogs.Resource.Attributes, "service.version").Value.StringValue != version.Version ||
		findAttribute(resourceLogs.Resource.Attributes, "host.id").Value.StringValue != hb.HostID ||
		findAttribute(resourceLogs.Resource.Attributes, "host.org").Value.StringValue != hb.HostOrg ||
		findAttribute(resourceLogs.Resource.Attributes, "host.boot_id").Value.StringValue != hb.BootID {
		t.Errorf("service or host resource attributes were not serialized for heartbeat %#v", hb)
	}

	scopeLogs := resourceLogs.ScopeLogs[0]
	if scopeLogs.Scope.Name != "github.com/m-horky/rhc-heartbeat" {
		t.Errorf("unexpected scope name %q", scopeLogs.Scope.Name)
	}

	record := scopeLogs.LogRecords[0]
	if record.TimeUnixNano != strconv.FormatInt(hb.TimeUnix.UnixNano(), 10) {
		t.Errorf("unexpected event timestamp %q", record.TimeUnixNano)
	}

	if record.ObservedTimeUnixNano != strconv.FormatInt(observedAt.UnixNano(), 10) {
		t.Errorf("unexpected observed timestamp %q", record.ObservedTimeUnixNano)
	}

	if record.SeverityText != "INFO" || record.Body.StringValue != "Heartbeat collected" {
		t.Errorf("unexpected log record: severity=%q body=%q", record.SeverityText, record.Body.StringValue)
	}

	monotonic := findAttribute(record.Attributes, "heartbeat.time_monotonic").Value.IntValue
	if findAttribute(record.Attributes, "heartbeat.trigger").Value.StringValue != string(hb.Trigger) ||
		monotonic != strconv.FormatInt(hb.TimeMonotonic.Nanoseconds(), 10) {
		t.Errorf("heartbeat record attributes were not serialized")
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
