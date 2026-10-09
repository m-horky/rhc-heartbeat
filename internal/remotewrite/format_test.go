package remotewrite

import (
	"testing"
	"time"

	"github.com/golang/snappy"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// TestEncodeHeartbeatsProducesSortedRemoteWriteSeries verifies the Remote Write v1 protobuf schema.
//
// Given heartbeat events sharing labels, when encoding runs, then it groups samples with sorted labels
// and Unix-millisecond timestamps.
func TestEncodeHeartbeatsProducesSortedRemoteWriteSeries(t *testing.T) {
	t.Parallel()

	heartbeats := []heartbeat.Heartbeat{
		{
			HostID: "uuid", HostOrg: "org", BootID: "boot", Kind: heartbeat.KindPing,
			TimeMonotonic: 3*time.Second + 500*time.Millisecond, TimeUnix: time.Unix(1700000001, 123456789),
		},
		{
			HostID: "uuid", HostOrg: "org", BootID: "boot", Kind: heartbeat.KindPing,
			TimeMonotonic: 2*time.Second + 500*time.Millisecond, TimeUnix: time.Unix(1700000000, 987654321),
		},
	}

	payload, err := encodeHeartbeats(heartbeats)
	if err != nil {
		t.Fatalf("encodeHeartbeats() error = %v", err)
	}

	schema := newRemoteWriteMessages()

	request := dynamicpb.NewMessage(schema.writeRequest)
	if err := proto.Unmarshal(payload, request); err != nil {
		t.Fatalf("unmarshal Remote Write protobuf: %v", err)
	}

	seriesList := request.Get(schema.writeRequest.Fields().ByName("timeseries")).List()
	if seriesList.Len() != 1 {
		t.Fatalf("Remote Write series count = %d, want 1", seriesList.Len())
	}

	series := seriesList.Get(0).Message()
	assertEncodedLabels(t, schema, series, heartbeat.KindPing)
	assertEncodedSamples(t, schema, series)
}

// TestEncodeHeartbeatsEmitsExtendedPayloadLabels verifies optional profile data reaches the Remote Write labels.
//
// Given a heartbeat with marketplace, CPU, and product details, when encoded, then those details are labeled
// and ID lists use comma-separated values.
func TestEncodeHeartbeatsEmitsExtendedPayloadLabels(t *testing.T) {
	t.Parallel()

	payload, err := encodeHeartbeats([]heartbeat.Heartbeat{{
		HostID: "uuid", HostOrg: "org", BootID: "boot", Kind: heartbeat.KindPing,
		MarketplaceID: "aws", MarketplaceAccountID: "account", MarketplaceInstanceID: "instance",
		MarketplaceOfferIDs: []string{"offer-a", "offer-b"}, VCPUCount: new(uint64(16)),
		ProductIDs: []string{"product-a", "product-b"},
	}})
	if err != nil {
		t.Fatalf("encodeHeartbeats() error = %v", err)
	}

	schema := newRemoteWriteMessages()

	request := dynamicpb.NewMessage(schema.writeRequest)
	if err := proto.Unmarshal(payload, request); err != nil {
		t.Fatalf("unmarshal Remote Write protobuf: %v", err)
	}

	series := request.Get(schema.writeRequest.Fields().ByName("timeseries")).List().Get(0).Message()
	labels := series.Get(schema.timeSeries.Fields().ByName("labels")).List()

	labelValues := make(map[string]string, labels.Len())
	for index := 0; index < labels.Len(); index++ {
		current := labels.Get(index).Message()
		name := current.Get(schema.label.Fields().ByName("name")).String()
		labelValues[name] = current.Get(schema.label.Fields().ByName("value")).String()
	}

	for name, want := range map[string]string{
		"marketplace_id":          "aws",
		"marketplace_account_id":  "account",
		"marketplace_instance_id": "instance",
		"marketplace_offer_ids":   "offer-a,offer-b",
		"vcpu_count":              "16",
		"product_ids":             "product-a,product-b",
	} {
		if labelValues[name] != want {
			t.Errorf("label %q = %q, want %q", name, labelValues[name], want)
		}
	}
}

// TestEncodeHeartbeatsEmitsSupportedKindLabels verifies each heartbeat enum reaches the Remote Write label unchanged.
//
// Given heartbeats with each supported kind
// When they are encoded
// Then each emitted kind label matches its enum value exactly.
func TestEncodeHeartbeatsEmitsSupportedKindLabels(t *testing.T) {
	t.Parallel()

	for _, kind := range []heartbeat.Kind{heartbeat.KindOn, heartbeat.KindOff, heartbeat.KindPing} {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()

			payload, err := encodeHeartbeats([]heartbeat.Heartbeat{{
				HostID: "uuid", HostOrg: "org", BootID: "boot", Kind: kind,
			}})
			if err != nil {
				t.Fatalf("encodeHeartbeats() error = %v", err)
			}

			schema := newRemoteWriteMessages()

			request := dynamicpb.NewMessage(schema.writeRequest)
			if err := proto.Unmarshal(payload, request); err != nil {
				t.Fatalf("unmarshal Remote Write protobuf: %v", err)
			}

			series := request.Get(schema.writeRequest.Fields().ByName("timeseries")).List().Get(0).Message()
			assertEncodedLabels(t, schema, series, kind)
		})
	}
}

// assertEncodedLabels checks the sorted label set and required heartbeat identity labels.
func assertEncodedLabels(
	t *testing.T,
	schema remoteWriteDescriptors,
	series protoreflect.Message,
	kind heartbeat.Kind,
) {
	t.Helper()

	labels := series.Get(schema.timeSeries.Fields().ByName("labels")).List()
	if labels.Len() != 5 {
		t.Fatalf("label count = %d, want 5", labels.Len())
	}

	labelValues := make(map[string]string, labels.Len())
	previousName := ""

	for i := 0; i < labels.Len(); i++ {
		current := labels.Get(i).Message()
		name := current.Get(schema.label.Fields().ByName("name")).String()

		labelValues[name] = current.Get(schema.label.Fields().ByName("value")).String()
		if name <= previousName {
			t.Errorf("label names are not sorted: %q follows %q", name, previousName)
		}

		previousName = name
	}

	for name, want := range map[string]string{
		"__name__":    "rhc_heartbeat_time_monotonic_seconds",
		"org_id":      "org",
		"system_uuid": "uuid",
		"boot_id":     "boot",
		"kind":        string(kind),
	} {
		if labelValues[name] != want {
			t.Errorf("label %q = %q, want %q", name, labelValues[name], want)
		}
	}
}

// assertEncodedSamples checks sample ordering, Unix-millisecond timestamps, and fractional monotonic seconds.
func assertEncodedSamples(t *testing.T, schema remoteWriteDescriptors, series protoreflect.Message) {
	t.Helper()

	samples := series.Get(schema.timeSeries.Fields().ByName("samples")).List()
	if samples.Len() != 2 {
		t.Fatalf("sample count = %d, want 2", samples.Len())
	}

	first := samples.Get(0).Message()
	second := samples.Get(1).Message()
	timestampField := schema.sample.Fields().ByName("timestamp")
	valueField := schema.sample.Fields().ByName("value")

	if got := first.Get(timestampField).Int(); got != 1700000000987 {
		t.Errorf("first sample timestamp = %d, want Unix milliseconds", got)
	}

	if got := second.Get(timestampField).Int(); got != 1700000001123 {
		t.Errorf("second sample timestamp = %d, want Unix milliseconds", got)
	}

	if got := first.Get(valueField).Float(); got != 2.5 {
		t.Errorf("first sample monotonic seconds = %v, want 2.5", got)
	}

	if got := second.Get(valueField).Float(); got != 3.5 {
		t.Errorf("second sample monotonic seconds = %v, want 3.5", got)
	}
}

// TestEncodeHeartbeatsRejectsEmptyRequiredLabelValues verifies invalid Remote Write labels are rejected.
//
// Given a heartbeat with a missing identity, when encoding runs, then it returns an error
// instead of emitting an invalid label.
func TestEncodeHeartbeatsRejectsEmptyRequiredLabelValues(t *testing.T) {
	t.Parallel()

	hb := heartbeat.Heartbeat{HostOrg: "org", BootID: "boot", Kind: heartbeat.KindPing}
	if _, err := encodeHeartbeats([]heartbeat.Heartbeat{hb}); err == nil {
		t.Fatal("encodeHeartbeats() error = nil, want empty label error")
	}
}

// TestEncodeHeartbeatsRejectsInvalidKind verifies unsupported kinds cannot become Prometheus labels.
//
// Given a heartbeat with a kind outside the declared enum
// When encoding runs
// Then encoding returns an error instead of emitting an unrestricted label.
func TestEncodeHeartbeatsRejectsInvalidKind(t *testing.T) {
	t.Parallel()

	hb := heartbeat.Heartbeat{
		HostID: "uuid", HostOrg: "org", BootID: "boot", Kind: heartbeat.Kind("custom"),
	}
	if _, err := encodeHeartbeats([]heartbeat.Heartbeat{hb}); err == nil {
		t.Fatal("encodeHeartbeats() error = nil, want invalid kind error")
	}
}

// TestSnappyPayloadUsesBlockEncoding verifies the Snappy block compressor accepts the protobuf bytes.
//
// Given an encoded Remote Write request, when block compression runs, then the payload decodes
// to the original protobuf bytes.
func TestSnappyPayloadUsesBlockEncoding(t *testing.T) {
	t.Parallel()

	hb := heartbeat.Heartbeat{HostID: "uuid", HostOrg: "org", BootID: "boot", Kind: heartbeat.KindOff}

	payload, err := encodeHeartbeats([]heartbeat.Heartbeat{hb})
	if err != nil {
		t.Fatalf("encodeHeartbeats() error = %v", err)
	}

	decoded, err := snappy.Decode(nil, snappy.Encode(nil, payload))
	if err != nil {
		t.Fatalf("snappy.Decode() error = %v", err)
	}

	if len(decoded) == 0 || len(decoded) != len(payload) {
		t.Errorf("decoded length = %d, protobuf length = %d", len(decoded), len(payload))
	}
}
