package profile

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/m-horky/rhc-heartbeat/internal/exec"
	"github.com/m-horky/rhc-heartbeat/internal/subman"
)

// Profile contains system profile and marketplace data.
type Profile struct {
	MarketplaceID         string
	MarketplaceAccountID  string
	MarketplaceInstanceID string
	MarketplaceOfferIDs   []string
	VCPUCount             *uint64
}

type provider struct {
	id          string
	instanceKey string
	accountKey  string
	offerKeys   []string
}

// providers returns cloud provider fact mappings in deterministic precedence order.
func providers() []provider {
	return []provider{
		{
			id:          "aws",
			instanceKey: "aws_instance_id",
			accountKey:  "aws_account_id",
			offerKeys:   []string{"aws_billing_products", "aws_marketplace_product_codes"},
		},
		{
			id:          "azure",
			instanceKey: "azure_instance_id",
			accountKey:  "azure_subscription_id",
			offerKeys:   []string{"azure_offer"},
		},
		{
			id:          "gcp",
			instanceKey: "gcp_instance_id",
			accountKey:  "gcp_project_number",
			offerKeys:   []string{"gcp_license_codes"},
		},
	}
}

// Get reads subscription-manager facts and transforms them into a system profile.
func Get() (Profile, error) {
	facts, err := subman.Read(exec.OSRunner{})
	if err != nil {
		return Profile{}, fmt.Errorf("read system profile facts: %w", err)
	}

	profile, err := extractFacts(facts)
	if err != nil {
		return Profile{}, fmt.Errorf("parse system profile facts: %w", err)
	}

	return profile, nil
}

// extractFacts derives system profile fields from subscription-manager facts.
func extractFacts(facts map[string]string) (Profile, error) {
	profile := Profile{}

	rawCount := strings.TrimSpace(facts["lscpu.cpu(s)"])
	if count, err := strconv.ParseUint(rawCount, 10, 64); err == nil && count > 0 {
		profile.VCPUCount = &count
	}

	for _, current := range providers() {
		if !hasProviderFacts(facts, current) {
			continue
		}

		profile.MarketplaceID = current.id
		profile.MarketplaceInstanceID = facts[current.instanceKey]
		profile.MarketplaceAccountID = facts[current.accountKey]
		profile.MarketplaceOfferIDs = collectOffers(facts, current.offerKeys)

		break
	}

	return profile, nil
}

// hasProviderFacts reports whether any fact with the provider's prefix is present.
func hasProviderFacts(facts map[string]string, current provider) bool {
	prefix := current.id + "_"
	for key := range facts {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}

	return false
}

// collectOffers combines comma-separated offer facts and removes duplicates.
func collectOffers(facts map[string]string, keys []string) []string {
	offerSet := make(map[string]struct{})

	for _, key := range keys {
		for offer := range strings.SplitSeq(facts[key], ",") {
			offer = strings.TrimSpace(offer)
			if offer != "" {
				offerSet[offer] = struct{}{}
			}
		}
	}

	offers := make([]string, 0, len(offerSet))
	for offer := range offerSet {
		offers = append(offers, offer)
	}

	sort.Strings(offers)

	return offers
}
