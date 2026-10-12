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
	BootID                string
	TimeUnix              time.Time
	TimeMonotonic         time.Duration
	Kind                  Kind
	HostID                *string
	HostOrg               *string
	MarketplaceID         *string
	MarketplaceAccountID  *string
	MarketplaceInstanceID *string
	MarketplaceOfferIDs   *[]string
	VCPUCount             *uint64
	ProductIDs            *[]string
}

// Get collects a heartbeat using the system defaults and the requested kind.
func Get(kind Kind) (Heartbeat, error) {
	hb, err := newHeartbeat(kind)
	if err != nil {
		return Heartbeat{}, fmt.Errorf("collect heartbeat: %w", err)
	}

	identity, err := consumer.Get()
	if err != nil {
		slog.Warn("cannot read heartbeat consumer identity", "err", err)
	} else {
		hb.HostID = new(identity.UUID)
		hb.HostOrg = new(identity.OrgID)
	}

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

// newHeartbeat creates a minimal Heartbeat object with boot and clock metadata.
func newHeartbeat(kind Kind) (Heartbeat, error) {
	bootID, err := bootid.Read(internalfs.Filesystem{})
	if err != nil {
		return Heartbeat{}, fmt.Errorf("construct a heartbeat: %w", err)
	}

	systemTime, err := clock.Read()
	if err != nil {
		return Heartbeat{}, fmt.Errorf("construct a heartbeat: %w", err)
	}

	return Heartbeat{
		BootID:        bootID,
		TimeUnix:      time.Unix(systemTime.TimeUnix.Sec, systemTime.TimeUnix.Nsec).UTC(),
		TimeMonotonic: time.Duration(systemTime.TimeMonotonic.Sec)*time.Second + time.Duration(systemTime.TimeMonotonic.Nsec),
		Kind:          kind,
	}, nil
}
