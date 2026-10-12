package remotewrite

import (
	"testing"
	"time"

	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
)

// TestSeriesOrder verifies time series are encoded chronologically.
//
// Given heartbeats from multiple boots supplied out of timestamp order, when encoding runs,
// then series are ordered by their earliest sample and samples are ordered within each series.
func TestSeriesOrder(t *testing.T) {
	t.Parallel()

	heartbeats := []heartbeat.Heartbeat{
		{
			HostID: new("uuid"), HostOrg: new("org"), BootID: "later", Kind: heartbeat.KindPing,
			TimeUnix: time.Unix(30, 0),
		},
		{
			HostID: new("uuid"), HostOrg: new("org"), BootID: "earlier", Kind: heartbeat.KindPing,
			TimeUnix: time.Unix(10, 0),
		},
		{
			HostID: new("uuid"), HostOrg: new("org"), BootID: "later", Kind: heartbeat.KindPing,
			TimeUnix: time.Unix(20, 0),
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
	if seriesList.Len() != 2 {
		t.Fatalf("Remote Write series count = %d, want 2", seriesList.Len())
	}

	want := []int64{10_000, 20_000}
	for index, expected := range want {
		series := seriesList.Get(index).Message()
		samples := series.Get(schema.timeSeries.Fields().ByName("samples")).List()

		first := samples.Get(0).Message().Get(schema.sample.Fields().ByName("timestamp")).Int()
		if first != expected {
			t.Errorf("series %d first timestamp = %d, want %d", index, first, expected)
		}
	}
}
