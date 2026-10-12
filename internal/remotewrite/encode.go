package remotewrite

import (
	"cmp"
	"fmt"
	"slices"
	"sort"

	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// newRemoteWriteMessages constructs descriptors for the stable Remote Write v1 protobuf schema.
func newRemoteWriteMessages() remoteWriteDescriptors {
	file, err := protodesc.NewFile(remoteWriteFileDescriptor, nil)
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

// encodeHeartbeats converts heartbeat records into sorted, grouped Remote Write time series.
func encodeHeartbeats(heartbeats []heartbeat.Heartbeat) ([]byte, error) {
	series, err := groupHeartbeats(heartbeats)
	if err != nil {
		return nil, err
	}

	return marshalSeries(newRemoteWriteMessages(), series)
}

// marshalSeries serializes grouped and ordered series as a Remote Write v1 WriteRequest.
func marshalSeries(schema remoteWriteDescriptors, seriesByKey map[uint64]*writeSeries) ([]byte, error) {
	keys := make([]uint64, 0, len(seriesByKey))
	for key := range seriesByKey {
		keys = append(keys, key)
	}

	slices.SortFunc(keys, cmp.Compare)

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
