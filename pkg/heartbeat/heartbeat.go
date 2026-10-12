package heartbeat

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/m-horky/rhc-heartbeat/internal/bootid"
	"github.com/m-horky/rhc-heartbeat/internal/clock"
	internalfs "github.com/m-horky/rhc-heartbeat/internal/fs"
	"github.com/m-horky/rhc-heartbeat/pkg/consumer"
	"github.com/m-horky/rhc-heartbeat/pkg/product"
	"github.com/m-horky/rhc-heartbeat/pkg/profile"
)

// Heartbeat contains system profile data and time.
type Heartbeat struct {
	HostID                string
	HostOrg               string
	BootID                string
	TimeUnix              time.Time
	TimeMonotonic         time.Duration
	Kind                  Kind
	MarketplaceID         *string
	MarketplaceAccountID  *string
	MarketplaceInstanceID *string
	MarketplaceOfferIDs   *[]string
	VCPUCount             *uint64
	ProductIDs            *[]string
}

// Get collects a heartbeat using the system defaults and the requested kind.
func Get(kind Kind) (Heartbeat, error) {
	// Go is making is handle the situations where some portion of the collection fails.
	// In real conditions, it should be very rare to impossible for it to happen.
	//
	// Given the situation, we are opting to include data telling us _the system
	// is online_, even if it is not billable because we don't know what it is.

	// Collect core metadata.
	hb, err := newHeartbeat(kind)
	if err != nil {
		return Heartbeat{}, fmt.Errorf("collect heartbeat: %w", err)
	}

	// Collect "optional" system profile.
	systemProfile, err := profile.Get()
	if err != nil {
		slog.Warn("cannot read heartbeat system profile", "err", err)
	} else {
		hb.MarketplaceID = systemProfile.ID
		hb.MarketplaceAccountID = systemProfile.Account
		hb.MarketplaceInstanceID = systemProfile.Instance
		hb.MarketplaceOfferIDs = systemProfile.Offers
		hb.VCPUCount = systemProfile.VCPUCount
	}

	// Collect "optional" system products.
	products, err := product.Get()
	if err != nil {
		slog.Warn("cannot read heartbeat product IDs", "err", err)
	} else {
		var productIDs []string
		for _, p := range products {
			productIDs = append(productIDs, strconv.Itoa(p.ID))
		}

		hb.ProductIDs = &productIDs
	}

	return hb, nil
}

// newHeartbeat creates a minimal Heartbeat object, containing required fields.
func newHeartbeat(kind Kind) (Heartbeat, error) {
	identity, err := consumer.Get()
	if err != nil {
		return Heartbeat{}, fmt.Errorf("construct a heartbeat: %w", err)
	}

	bootID, err := bootid.Read(internalfs.Filesystem{})
	if err != nil {
		return Heartbeat{}, fmt.Errorf("construct a heartbeat: %w", err)
	}

	systemTime, err := clock.Read()
	if err != nil {
		return Heartbeat{}, fmt.Errorf("construct a heartbeat: %w", err)
	}

	return Heartbeat{
		HostID:        identity.UUID,
		HostOrg:       identity.OrgID,
		BootID:        bootID,
		TimeUnix:      time.Unix(systemTime.TimeUnix.Sec, systemTime.TimeUnix.Nsec).UTC(),
		TimeMonotonic: time.Duration(systemTime.TimeMonotonic.Sec)*time.Second + time.Duration(systemTime.TimeMonotonic.Nsec),
		Kind:          kind,
	}, nil
}
