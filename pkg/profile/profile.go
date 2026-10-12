package profile

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/m-horky/rhc-heartbeat/internal/exec"
	"github.com/m-horky/rhc-heartbeat/internal/subman"
)

type marketplaceProfile struct {
	ID       *string
	Account  *string
	Instance *string
	Offers   *[]string
}

type cpuProfile struct {
	VCPUCount *uint64
}

// Profile contains system profile and marketplace data.
type Profile struct {
	marketplaceProfile
	cpuProfile
}

// Get reads subscription-manager facts and transforms them into a system profile.
func Get() (Profile, error) {
	facts, err := subman.GetFacts(exec.OSRunner{})
	if err != nil {
		return Profile{}, fmt.Errorf("read system profile facts: %w", err)
	}

	profile := Profile{}

	vCPU, err := newCPUProfile(facts)
	if err != nil {
		slog.Warn("cannot extract vCPU count", "err", err)
	}

	profile.VCPUCount = vCPU.VCPUCount

	marketplace, err := newMarketplaceProfile(facts)
	if err != nil {
		slog.Debug("cannot extract marketplace profile", "err", err)
	}

	profile.ID = marketplace.ID
	profile.Instance = marketplace.Instance
	profile.Account = marketplace.Account
	profile.Offers = marketplace.Offers

	return profile, nil
}

// newCPUProfile reads logical CPU count from subscription-manager facts output.
func newCPUProfile(facts map[string]string) (*cpuProfile, error) {
	raw, ok := facts["lscpu.cpu(s)"]
	if !ok {
		return nil, errors.New("lscpu.cpu(s) absent")
	}

	count, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse lscpu.cpu(s): %w", err)
	}

	return &cpuProfile{VCPUCount: &count}, nil
}

// newMarketplaceProfile derives system profile fields from subscription-manager facts.
func newMarketplaceProfile(facts map[string]string) (*marketplaceProfile, error) {
	profile := &marketplaceProfile{}

	var (
		id       string
		instance string
		account  string
		offers   string
	)

	if _, ok := facts["aws_instance_id"]; ok {
		id = "aws"
		instance = facts["aws_instance_id"]
		account = facts["aws_account_id"]
		offers = facts["aws_billing_products"] + facts["aws_marketplace_product_codes"]
	}

	if _, ok := facts["azure_instance_id"]; ok {
		id = "azure"
		instance = facts["azure_instance_id"]
		account = facts["azure_subscription_id"]
		offers = facts["azure_offer"]
	}

	if _, ok := facts["gcp_instance_id"]; ok {
		id = "gcp"
		instance = facts["gcp_instance_id"]
		account = facts["gcp_project_number"]
		offers = facts["gcp_license_codes"]
	}

	// The concatenation above may have resulted in a string starting/ending with a comma.
	var offersList []string

	for offer := range strings.SplitSeq(offers, ",") {
		if len(offer) == 0 {
			continue
		}

		offersList = append(offersList, offer)
	}

	if id != "" {
		profile.ID = new(id)
	}

	if instance != "" {
		profile.Instance = new(instance)
	}

	if account != "" {
		profile.Account = new(account)
	}

	if len(offersList) > 0 {
		profile.Offers = &offersList
	}

	return profile, nil
}
