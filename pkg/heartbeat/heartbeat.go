package heartbeat

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/m-horky/rhc-heartbeat/internal/bootid"
	"github.com/m-horky/rhc-heartbeat/internal/clock"
	"github.com/m-horky/rhc-heartbeat/internal/fs"
	"github.com/m-horky/rhc-heartbeat/pkg/consumer"
	"github.com/m-horky/rhc-heartbeat/pkg/products"
	"github.com/m-horky/rhc-heartbeat/pkg/profile"
)

// Heartbeat contains system profile data and time.
type Heartbeat struct {
	HostID                string
	HostOrg               string
	BootID                string
	TimeMonotonic         time.Duration
	TimeBoottime          time.Duration
	TimeUnix              time.Time
	Kind                  Kind
	MarketplaceID         string
	MarketplaceAccountID  string
	MarketplaceInstanceID string
	MarketplaceOfferIDs   []string
	VCPUCount             *uint64
	ProductIDs            []string
}

// sources groups heartbeat inputs so collect can be tested with deterministic readers
// while Get uses system implementations.
type sources struct {
	readIdentity   func() (consumer.Identity, error)
	readBootID     func() (string, error)
	readClock      func() (clock.Reading, error)
	readProfile    func() (profile.Profile, error)
	readProductIDs func() ([]string, error)
}

// Get collects a heartbeat using the system defaults and the requested kind.
func Get(ctx context.Context, kind Kind) (Heartbeat, error) {
	return collect(ctx, kind, sources{
		readIdentity: consumer.Get,
		readBootID: func() (string, error) {
			return bootid.Read(fs.Filesystem{})
		},
		readClock:      clock.Read,
		readProfile:    profile.Get,
		readProductIDs: products.Get,
	})
}

// collect reads each heartbeat source and combines the results into one value.
func collect(ctx context.Context, kind Kind, source sources) (Heartbeat, error) {
	if !kind.Valid() {
		return Heartbeat{}, fmt.Errorf("collect heartbeat: invalid kind %q", kind)
	}

	if err := ctx.Err(); err != nil {
		return Heartbeat{}, fmt.Errorf("collect heartbeat: %w", err)
	}

	identity, err := source.readIdentity()
	if err != nil {
		return Heartbeat{}, fmt.Errorf("read heartbeat identity: %w", err)
	}

	bootID, err := source.readBootID()
	if err != nil {
		return Heartbeat{}, fmt.Errorf("read heartbeat boot ID: %w", err)
	}

	clockReading, err := source.readClock()
	if err != nil {
		return Heartbeat{}, fmt.Errorf("read heartbeat clocks: %w", err)
	}

	systemProfile, err := source.readProfile()
	if err != nil {
		slog.Warn("cannot read heartbeat system profile", "err", err)

		systemProfile = profile.Profile{}
	}

	productIDs, err := source.readProductIDs()
	if err != nil {
		slog.Warn("cannot read heartbeat product IDs", "err", err)

		productIDs = nil
	}

	monotonicTime := time.Duration(clockReading.TimeMonotonic.Sec)*time.Second +
		time.Duration(clockReading.TimeMonotonic.Nsec)
	boottime := time.Duration(clockReading.TimeBoottime.Sec)*time.Second +
		time.Duration(clockReading.TimeBoottime.Nsec)
	unixTime := time.Unix(clockReading.Time.Sec, clockReading.Time.Nsec).UTC()

	return Heartbeat{
		HostID:                identity.UUID,
		HostOrg:               identity.OrgID,
		BootID:                bootID,
		TimeMonotonic:         monotonicTime,
		TimeBoottime:          boottime,
		TimeUnix:              unixTime,
		Kind:                  kind,
		MarketplaceID:         systemProfile.MarketplaceID,
		MarketplaceAccountID:  systemProfile.MarketplaceAccountID,
		MarketplaceInstanceID: systemProfile.MarketplaceInstanceID,
		MarketplaceOfferIDs:   systemProfile.MarketplaceOfferIDs,
		VCPUCount:             systemProfile.VCPUCount,
		ProductIDs:            productIDs,
	}, nil
}
