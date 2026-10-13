package remotewrite

import (
	"cmp"
	"fmt"
	"hash/fnv"
	"slices"
	"strconv"
	"strings"

	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
)

// This package should be refactored, with heartbeat-specific code living elsewhere.
// Since we currently have only a single Prometheus series, it does not matter.

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

// groupHeartbeats validates labels and groups event samples by their complete label set.
func groupHeartbeats(heartbeats []heartbeat.Heartbeat) (map[uint64]*writeSeries, error) {
	series := make(map[uint64]*writeSeries)

	// Group samples into series.
	// Prometheus expects a single TimeSeries containing all collected values
	// for that label set. In practice, we will almost always have created
	// a single series, since the label values are fairly static.
	for _, hb := range heartbeats {
		labels, err := newLabelsFromHeartbeat(hb)
		if err != nil {
			return nil, fmt.Errorf("error formatting heartbeats: %w", err)
		}

		key := newKeyFromLabels(labels)

		hbSeries := series[key]
		if hbSeries == nil {
			hbSeries = &writeSeries{labels: labels}
			series[key] = hbSeries
		}

		hbSeries.samples = append(hbSeries.samples, sample{
			value:     hb.TimeMonotonic.Seconds(),
			timestamp: hb.TimeUnix.UnixMilli(),
		})
	}

	return series, nil
}

// newLabelsFromHeartbeat returns labels describing one heartbeat.
func newLabelsFromHeartbeat(hb heartbeat.Heartbeat) ([]label, error) {
	if hb.HostID == nil || *hb.HostID == "" {
		return nil, fmt.Errorf("missing host_uuid")
	}

	if hb.HostOrg == nil || *hb.HostOrg == "" {
		return nil, fmt.Errorf("missing host_org")
	}

	labels := []label{
		{name: "__name__", value: "rhc_heartbeat_time_monotonic_seconds"},
		{name: "host_uuid", value: *hb.HostID},
		{name: "host_org", value: *hb.HostOrg},
		{name: "boot_id", value: hb.BootID},
		{name: "kind", value: string(hb.Kind)},
	}

	if hb.MarketplaceID != nil {
		labels = append(labels, label{name: "marketplace_id", value: *hb.MarketplaceID})
	}

	if hb.MarketplaceAccountID != nil {
		labels = append(labels, label{name: "marketplace_account_id", value: *hb.MarketplaceAccountID})
	}

	if hb.MarketplaceInstanceID != nil {
		labels = append(labels, label{name: "marketplace_instance_id", value: *hb.MarketplaceInstanceID})
	}

	if hb.MarketplaceOfferIDs != nil {
		labels = append(labels, label{name: "marketplace_offer_ids", value: strings.Join(*hb.MarketplaceOfferIDs, ",")})
	}

	if hb.VCPUCount != nil && *hb.VCPUCount > 0 {
		labels = append(labels, label{name: "vcpu_count", value: strconv.FormatUint(*hb.VCPUCount, 10)})
	}

	if hb.ProductIDs != nil {
		labels = append(labels, label{name: "product_ids", value: strings.Join(*hb.ProductIDs, ",")})
	}

	slices.SortFunc(labels, func(a, b label) int { return cmp.Compare(a.name, b.name) })

	return labels, nil
}

// newKeyFromLabels returns a static hash for a supplied key=value set.
func newKeyFromLabels(labels []label) uint64 {
	h := fnv.New64a()
	for _, l := range labels {
		_, _ = h.Write([]byte(l.name))
		_, _ = h.Write([]byte(l.value))
	}

	return h.Sum64()
}
