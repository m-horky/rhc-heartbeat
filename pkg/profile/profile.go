package profile

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/m-horky/rhc-heartbeat/internal/exec"
	"github.com/m-horky/rhc-heartbeat/internal/subman"
	"github.com/m-horky/rhc-heartbeat/pkg/constants"
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
	facts, err := subman.GetFacts(exec.OSRunner{}, constants.Path(constants.RHSMBinPath))
	if err != nil {
		return Profile{}, fmt.Errorf("read system profile facts: %w", err)
	}

	profile := Profile{}

	cpu := newCPUProfile(facts)
	marketplace := newMarketplaceProfile(facts)

	profile.VCPUCount = cpu.VCPUCount
	profile.ID = marketplace.ID
	profile.Instance = marketplace.Instance
	profile.Account = marketplace.Account
	profile.Offers = marketplace.Offers

	return profile, nil
}

// newCPUProfile reads logical CPU count from subscription-manager facts output.
func newCPUProfile(facts map[string]string) cpuProfile {
	raw, ok := facts["lscpu.cpu(s)"]
	if !ok {
		slog.Warn("cannot extract vCPU count", "err", "lscpu.cpu(s) absent")

		return cpuProfile{VCPUCount: nil}
	}

	count, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		slog.Warn("cannot extract vCPU count", "err", err)

		return cpuProfile{VCPUCount: nil}
	}

	return cpuProfile{VCPUCount: &count}
}

// newMarketplaceProfile derives system profile fields from subscription-manager facts.
func newMarketplaceProfile(facts map[string]string) marketplaceProfile {
	profile := marketplaceProfile{}

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
		offers = strings.Join([]string{
			facts["aws_billing_products"],
			facts["aws_marketplace_product_codes"],
		}, ",")
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

	offersList := parseOffers(offers)

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

	return profile
}

// parseOffers splits, trims, and deduplicates comma-separated marketplace offers.
func parseOffers(offers string) []string {
	var offersList []string

	seenOffers := make(map[string]struct{})

	for offer := range strings.SplitSeq(offers, ",") {
		offer = strings.TrimSpace(offer)
		if offer == "" {
			continue
		}

		if _, seen := seenOffers[offer]; seen {
			continue
		}

		seenOffers[offer] = struct{}{}
		offersList = append(offersList, offer)
	}

	return offersList
}
