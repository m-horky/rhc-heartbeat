// Package remotewrite encodes and uploads heartbeat samples using Prometheus Remote Write 1.0.
package remotewrite

import (
	"fmt"
	"sort"
	"strings"

	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

type remoteWriteDescriptors struct {
	writeRequest protoreflect.MessageDescriptor
	timeSeries   protoreflect.MessageDescriptor
	label        protoreflect.MessageDescriptor
	sample       protoreflect.MessageDescriptor
}

type writeSeries struct {
	labels  []label
	samples []sample
}

type label struct {
	name  string
	value string
}

type sample struct {
	value     float64
	timestamp int64
}

// newRemoteWriteMessages constructs descriptors for the stable Remote Write v1 protobuf schema.
func newRemoteWriteMessages() remoteWriteDescriptors {
	file, err := protodesc.NewFile(remoteWriteFileDescriptor(), nil)
	if err != nil {
		panic(fmt.Sprintf("define Prometheus Remote Write protobuf schema: %v", err))
	}

	messages := file.Messages()

	return remoteWriteDescriptors{
		writeRequest: messages.ByName("WriteRequest"),
		timeSeries:   messages.ByName("TimeSeries"),
		label:        messages.ByName("Label"),
		sample:       messages.ByName("Sample"),
	}
}

// remoteWriteFileDescriptor describes the Remote Write v1 fields using protobuf descriptors.
func remoteWriteFileDescriptor() *descriptorpb.FileDescriptorProto {
	optional := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	repeated := descriptorpb.FieldDescriptorProto_LABEL_REPEATED
	stringType := descriptorpb.FieldDescriptorProto_TYPE_STRING
	doubleType := descriptorpb.FieldDescriptorProto_TYPE_DOUBLE
	int64Type := descriptorpb.FieldDescriptorProto_TYPE_INT64
	messageType := descriptorpb.FieldDescriptorProto_TYPE_MESSAGE

	return &descriptorpb.FileDescriptorProto{
		Name:    new("prometheus/write_request.proto"),
		Package: new("prometheus"),
		Syntax:  new("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: new("WriteRequest"),
				ReservedRange: []*descriptorpb.DescriptorProto_ReservedRange{
					{Start: proto.Int32(2), End: proto.Int32(4)},
				},
				Field: []*descriptorpb.FieldDescriptorProto{
					remoteWriteField("timeseries", 1, repeated, messageType, ".prometheus.TimeSeries"),
				},
			},
			{
				Name: new("TimeSeries"),
				Field: []*descriptorpb.FieldDescriptorProto{
					remoteWriteField("labels", 1, repeated, messageType, ".prometheus.Label"),
					remoteWriteField("samples", 2, repeated, messageType, ".prometheus.Sample"),
				},
			},
			{
				Name: new("Label"),
				Field: []*descriptorpb.FieldDescriptorProto{
					remoteWriteField("name", 1, optional, stringType, ""),
					remoteWriteField("value", 2, optional, stringType, ""),
				},
			},
			{
				Name: new("Sample"),
				Field: []*descriptorpb.FieldDescriptorProto{
					remoteWriteField("value", 1, optional, doubleType, ""),
					remoteWriteField("timestamp", 2, optional, int64Type, ""),
				},
			},
		},
	}
}

// remoteWriteField creates one field descriptor for the Remote Write protobuf schema.
func remoteWriteField(
	name string,
	number int32,
	label descriptorpb.FieldDescriptorProto_Label,
	kind descriptorpb.FieldDescriptorProto_Type,
	typeName string,
) *descriptorpb.FieldDescriptorProto {
	value := &descriptorpb.FieldDescriptorProto{
		Name:   new(name),
		Number: new(number),
		Label:  &label,
		Type:   &kind,
	}
	if typeName != "" {
		value.TypeName = new(typeName)
	}

	return value
}

// encodeHeartbeats converts heartbeat records into sorted, grouped Remote Write time series.
func encodeHeartbeats(heartbeats []heartbeat.Heartbeat) ([]byte, error) {
	series, err := groupHeartbeats(heartbeats)
	if err != nil {
		return nil, err
	}

	return marshalSeries(newRemoteWriteMessages(), series)
}

// groupHeartbeats validates labels and groups event samples by their complete label set.
func groupHeartbeats(heartbeats []heartbeat.Heartbeat) (map[string]*writeSeries, error) {
	seriesByKey := make(map[string]*writeSeries)

	for _, hb := range heartbeats {
		if !hb.Kind.Valid() {
			return nil, fmt.Errorf("encode heartbeat: invalid kind %q", hb.Kind)
		}

		labels := heartbeatLabels(hb)
		for _, current := range labels {
			if current.value == "" {
				return nil, fmt.Errorf("encode heartbeat: label %q has an empty value", current.name)
			}
		}

		key := labelSetKey(labels)

		series := seriesByKey[key]
		if series == nil {
			series = &writeSeries{labels: labels}
			seriesByKey[key] = series
		}

		series.samples = append(series.samples, sample{
			value:     hb.TimeMonotonic.Seconds(),
			timestamp: hb.TimeUnix.UnixMilli(),
		})
	}

	return seriesByKey, nil
}

// heartbeatLabels returns the lexicographically sorted labels describing one heartbeat series.
func heartbeatLabels(hb heartbeat.Heartbeat) []label {
	labels := []label{
		{name: "__name__", value: "rhc_heartbeat_time_monotonic_seconds"},
		{name: "boot_id", value: hb.BootID},
		{name: "kind", value: string(hb.Kind)},
		{name: "org_id", value: hb.HostOrg},
		{name: "system_uuid", value: hb.HostID},
	}
	sort.Slice(labels, func(i, j int) bool { return labels[i].name < labels[j].name })

	return labels
}

// labelSetKey returns an unambiguous key for a sorted set of labels.
func labelSetKey(labels []label) string {
	var key strings.Builder
	for _, current := range labels {
		_, _ = fmt.Fprintf(&key, "%d:%s%d:%s", len(current.name), current.name, len(current.value), current.value)
	}

	return key.String()
}

// marshalSeries serializes grouped and ordered series as a Remote Write v1 WriteRequest.
func marshalSeries(schema remoteWriteDescriptors, seriesByKey map[string]*writeSeries) ([]byte, error) {
	keys := make([]string, 0, len(seriesByKey))
	for key := range seriesByKey {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	request := dynamicpb.NewMessage(schema.writeRequest)
	field := schema.writeRequest.Fields().ByName("timeseries")

	seriesList := request.Mutable(field).List()
	for _, key := range keys {
		appendSeries(schema, seriesList, seriesByKey[key])
	}

	payload, err := proto.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal Prometheus Remote Write request: %w", err)
	}

	return payload, nil
}

// appendSeries adds one label set and its timestamp-ordered samples to the request.
func appendSeries(schema remoteWriteDescriptors, seriesList protoreflect.List, series *writeSeries) {
	sort.SliceStable(series.samples, func(i, j int) bool {
		return series.samples[i].timestamp < series.samples[j].timestamp
	})

	message := dynamicpb.NewMessage(schema.timeSeries)
	appendLabels(schema, message, series.labels)
	appendSamples(schema, message, series.samples)
	seriesList.Append(protoreflect.ValueOfMessage(message))
}

// appendLabels adds sorted label pairs to a Remote Write time series message.
func appendLabels(schema remoteWriteDescriptors, series protoreflect.Message, labels []label) {
	field := schema.timeSeries.Fields().ByName("labels")
	list := series.Mutable(field).List()

	for _, current := range labels {
		message := dynamicpb.NewMessage(schema.label)
		message.Set(schema.label.Fields().ByName("name"), protoreflect.ValueOfString(current.name))
		message.Set(schema.label.Fields().ByName("value"), protoreflect.ValueOfString(current.value))
		list.Append(protoreflect.ValueOfMessage(message))
	}
}

// appendSamples adds ordered sample values and timestamps to a Remote Write time series message.
func appendSamples(schema remoteWriteDescriptors, series protoreflect.Message, samples []sample) {
	field := schema.timeSeries.Fields().ByName("samples")
	list := series.Mutable(field).List()

	for _, current := range samples {
		message := dynamicpb.NewMessage(schema.sample)
		message.Set(schema.sample.Fields().ByName("value"), protoreflect.ValueOfFloat64(current.value))
		message.Set(
			schema.sample.Fields().ByName("timestamp"),
			protoreflect.ValueOfInt64(current.timestamp),
		)
		list.Append(protoreflect.ValueOfMessage(message))
	}
}
