package heartbeat

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/m-horky/rhc-heartbeat/internal/clock"
	"github.com/m-horky/rhc-heartbeat/pkg/consumer"
	"github.com/m-horky/rhc-heartbeat/pkg/profile"
	"golang.org/x/sys/unix"
)

// TestCollectAssemblesHeartbeat verifies successful source readings populate the heartbeat.
//
// Given successful identity, boot ID, and clock sources, when collecting,
// then all readings and the heartbeat kind are returned.
func TestCollectAssemblesHeartbeat(t *testing.T) {
	t.Parallel()

	monotonic := unix.Timespec{Sec: 42, Nsec: 123}
	boottime := unix.Timespec{Sec: 45, Nsec: 789}
	realtime := unix.Timespec{Sec: 1_741_000_000, Nsec: 456}
	source := testSources()
	source.readIdentity = func() (consumer.Identity, error) {
		return consumer.Identity{UUID: "system-uuid", OrgID: "org-id"}, nil
	}
	source.readBootID = func() (string, error) { return "boot-id", nil }
	source.readClock = func() (clock.Reading, error) {
		return clock.Reading{TimeMonotonic: monotonic, TimeBoottime: boottime, Time: realtime}, nil
	}

	got, err := collect(context.Background(), KindOff, source)
	if err != nil {
		t.Fatalf("collect() error = %v", err)
	}

	if got.HostID != "system-uuid" {
		t.Errorf("HostID = %q, want %q", got.HostID, "system-uuid")
	}

	if got.HostOrg != "org-id" {
		t.Errorf("HostOrg = %q, want %q", got.HostOrg, "org-id")
	}

	if got.BootID != "boot-id" {
		t.Errorf("BootID = %q, want %q", got.BootID, "boot-id")
	}

	wantMonotonic := time.Duration(monotonic.Sec)*time.Second + time.Duration(monotonic.Nsec)
	if got.TimeMonotonic != wantMonotonic {
		t.Errorf("TimeMonotonic = %s, want %s", got.TimeMonotonic, wantMonotonic)
	}

	wantBoottime := time.Duration(boottime.Sec)*time.Second + time.Duration(boottime.Nsec)
	if got.TimeBoottime != wantBoottime {
		t.Errorf("TimeBoottime = %s, want %s", got.TimeBoottime, wantBoottime)
	}

	wantUnix := time.Unix(realtime.Sec, realtime.Nsec).UTC()
	if !got.TimeUnix.Equal(wantUnix) {
		t.Errorf("TimeUnix = %s, want %s", got.TimeUnix, wantUnix)
	}

	if got.Kind != KindOff {
		t.Errorf("Kind = %q, want %q", got.Kind, KindOff)
	}
}

// TestCollectIncludesExtendedProfile verifies marketplace, CPU, and product data are carried into a heartbeat.
//
// Given successful profile and product sources, when collecting, then every extended field is preserved.
func TestCollectIncludesExtendedProfile(t *testing.T) {
	t.Parallel()

	source := testSources()
	source.readProfile = func() (profile.Profile, error) {
		return profile.Profile{
			MarketplaceID: "aws", MarketplaceAccountID: "account-id", MarketplaceInstanceID: "instance-id",
			MarketplaceOfferIDs: []string{"offer-a", "offer-b"}, VCPUCount: new(uint64(8)),
		}, nil
	}
	source.readProductIDs = func() ([]string, error) { return []string{"product-a", "product-b"}, nil }

	got, err := collect(context.Background(), KindPing, source)
	if err != nil {
		t.Fatalf("collect() error = %v", err)
	}

	if got.MarketplaceID != "aws" || got.MarketplaceAccountID != "account-id" ||
		got.MarketplaceInstanceID != "instance-id" {
		t.Errorf("marketplace identity = %#v, want AWS account and instance details", got)
	}

	if !slices.Equal(got.MarketplaceOfferIDs, []string{"offer-a", "offer-b"}) {
		t.Errorf("MarketplaceOfferIDs = %#v, want [offer-a offer-b]", got.MarketplaceOfferIDs)
	}

	if got.VCPUCount == nil || *got.VCPUCount != 8 {
		t.Errorf("VCPUCount = %v, want 8", got.VCPUCount)
	}

	if !slices.Equal(got.ProductIDs, []string{"product-a", "product-b"}) {
		t.Errorf("ProductIDs = %#v, want [product-a product-b]", got.ProductIDs)
	}
}

// TestKindValid verifies the heartbeat kind enum accepts only its declared values.
//
// Given each declared kind and an unsupported string
// When validity is checked
// Then only on, off, and ping are accepted.
func TestKindValid(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		kind Kind
		want bool
	}{
		{kind: KindOn, want: true},
		{kind: KindOff, want: true},
		{kind: KindPing, want: true},
		{kind: Kind("custom"), want: false},
		{kind: Kind(""), want: false},
	} {
		if got := test.kind.Valid(); got != test.want {
			t.Errorf("Kind(%q).Valid() = %t, want %t", test.kind, got, test.want)
		}
	}
}

// TestCollectRejectsInvalidKind verifies collection rejects kinds outside the declared enum.
//
// Given successful heartbeat sources and an unsupported kind
// When a heartbeat is collected
// Then collection fails before returning a heartbeat.
func TestCollectRejectsInvalidKind(t *testing.T) {
	t.Parallel()

	if _, err := collect(context.Background(), Kind("custom"), testSources()); err == nil {
		t.Fatal("collect() error = nil, want invalid kind error")
	}
}

// TestCollectPropagatesContextCancellation verifies canceled collection contexts are respected.
//
// Given an already canceled context, when collecting a heartbeat, then the cancellation is returned.
func TestCollectPropagatesContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := collect(ctx, KindPing, testSources())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("collect() error = %v, want context.Canceled", err)
	}
}

// TestCollectWrapsSourceErrors verifies collection stops and identifies the failing source.
//
// Given any heartbeat source fails, when collecting, then its failure is returned with source context.
func TestCollectWrapsSourceErrors(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("source unavailable")
	tests := []struct {
		name        string
		configure   func(*sources)
		wantContext string
	}{
		{
			name: "identity",
			configure: func(source *sources) {
				source.readIdentity = func() (consumer.Identity, error) { return consumer.Identity{}, wantErr }
			},
			wantContext: "read heartbeat identity",
		},
		{
			name: "boot ID",
			configure: func(source *sources) {
				source.readBootID = func() (string, error) { return "", wantErr }
			},
			wantContext: "read heartbeat boot ID",
		},
		{
			name: "clock",
			configure: func(source *sources) {
				source.readClock = func() (clock.Reading, error) { return clock.Reading{}, wantErr }
			},
			wantContext: "read heartbeat clocks",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			source := testSources()
			test.configure(&source)

			_, err := collect(context.Background(), KindPing, source)
			if !errors.Is(err, wantErr) {
				t.Fatalf("collect() error = %v, want wrapped %v", err, wantErr)
			}

			if !strings.Contains(err.Error(), test.wantContext) {
				t.Errorf("collect() error = %v, want context %q", err, test.wantContext)
			}
		})
	}
}

// TestCollectContinuesWhenProfileReadFails verifies system profile data is optional enrichment.
//
// Given the system profile source fails, when collecting, then the heartbeat is returned without profile data.
func TestCollectContinuesWhenProfileReadFails(t *testing.T) {
	t.Parallel()

	source := testSources()
	source.readProfile = func() (profile.Profile, error) { return profile.Profile{}, errors.New("source unavailable") }

	got, err := collect(context.Background(), KindPing, source)
	if err != nil {
		t.Fatalf("collect() error = %v, want successful heartbeat without profile data", err)
	}

	if got.VCPUCount != nil || got.MarketplaceID != "" {
		t.Errorf("profile fields = %#v, want empty values after profile read failure", got)
	}

	if got.HostID != "test-uuid" || got.Kind != KindPing {
		t.Errorf("heartbeat = %#v, want collected identity and kind", got)
	}
}

// TestCollectContinuesWhenProductIDReadFails verifies product IDs are optional enrichment.
//
// Given product ID reading fails, when collecting, then the heartbeat is returned without product IDs.
func TestCollectContinuesWhenProductIDReadFails(t *testing.T) {
	t.Parallel()

	source := testSources()
	source.readProductIDs = func() ([]string, error) { return nil, errors.New("certificate unavailable") }

	got, err := collect(context.Background(), KindPing, source)
	if err != nil {
		t.Fatalf("collect() error = %v, want successful heartbeat without product IDs", err)
	}

	if got.ProductIDs != nil {
		t.Errorf("ProductIDs = %#v, want nil when product ID reading fails", got.ProductIDs)
	}

	if got.HostID != "test-uuid" || got.Kind != KindPing {
		t.Errorf("heartbeat = %#v, want collected identity and kind", got)
	}
}

// testSources returns successful heartbeat sources for collection tests.
func testSources() sources {
	return sources{
		readIdentity: func() (consumer.Identity, error) {
			return consumer.Identity{UUID: "test-uuid", OrgID: "test-org"}, nil
		},
		readBootID: func() (string, error) {
			return "test-boot-id", nil
		},
		readClock: func() (clock.Reading, error) {
			return clock.Reading{
				TimeMonotonic: unix.Timespec{Sec: 1, Nsec: 2},
				TimeBoottime:  unix.Timespec{Sec: 1, Nsec: 4},
				Time:          unix.Timespec{Sec: 1, Nsec: 3},
			}, nil
		},
		readProfile:    func() (profile.Profile, error) { return profile.Profile{VCPUCount: new(uint64(2))}, nil },
		readProductIDs: func() ([]string, error) { return []string{}, nil },
	}
}
