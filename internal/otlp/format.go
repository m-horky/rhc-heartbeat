// Package otlp formats heartbeats as OTLP logs and uploads them over OTLP/HTTP.
package otlp

import (
	"time"

	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
	"github.com/m-horky/rhc-heartbeat/pkg/version"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

// buildLogs maps heartbeats to OTLP log records observed at observedAt.
func buildLogs(heartbeats []heartbeat.Heartbeat, observedAt time.Time) plog.Logs {
	logs := plog.NewLogs()

	for _, hb := range heartbeats {
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
	}

	return logs
}
