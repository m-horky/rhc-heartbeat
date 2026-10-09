package profile

import (
	"slices"
	"testing"
)

// TestParseDerivesMarketplaceAndVCPU verifies AWS fields and CPU count are transformed from facts.
//
// Given AWS marketplace facts and a CPU fact, when parsed,
// then the AWS identifiers, unique offers, and vCPU count are returned.
func TestParseDerivesMarketplaceAndVCPU(t *testing.T) {
	t.Parallel()

	got, err := extractFacts(map[string]string{
		"lscpu.cpu(s)":                  " 8 ",
		"aws_instance_id":               "i-123",
		"aws_account_id":                "123456789012",
		"aws_billing_products":          "bp-1, bp-2",
		"aws_marketplace_product_codes": "product-1,bp-2",
	})
	if err != nil {
		t.Fatalf("extractFacts() error = %v", err)
	}

	if got.MarketplaceID != "aws" || got.MarketplaceInstanceID != "i-123" || got.MarketplaceAccountID != "123456789012" {
		t.Errorf("extractFacts() identity = %#v, want AWS marketplace identity", got)
	}

	if !equalStringsIgnoringOrder(got.MarketplaceOfferIDs, []string{"bp-1", "bp-2", "product-1"}) {
		t.Errorf("extractFacts() OfferIDs = %#v, want [bp-1 bp-2 product-1] in any order", got.MarketplaceOfferIDs)
	}

	if got.VCPUCount == nil || *got.VCPUCount != 8 {
		t.Errorf("extractFacts() VCPUCount = %v, want 8", got.VCPUCount)
	}
}

// TestParseSortsMarketplaceOfferIDs verifies marketplace offer values have deterministic order.
//
// Given marketplace offer facts in arbitrary order, when parsed, then unique offers are returned sorted.
func TestParseSortsMarketplaceOfferIDs(t *testing.T) {
	t.Parallel()

	got, err := extractFacts(map[string]string{
		"lscpu.cpu(s)":                  "4",
		"aws_billing_products":          "offer-z, offer-a",
		"aws_marketplace_product_codes": "offer-m,offer-a",
	})
	if err != nil {
		t.Fatalf("extractFacts() error = %v", err)
	}

	want := []string{"offer-a", "offer-m", "offer-z"}
	if !slices.Equal(got.MarketplaceOfferIDs, want) {
		t.Errorf("MarketplaceOfferIDs = %#v, want %#v", got.MarketplaceOfferIDs, want)
	}
}

// TestParseMapsAzureAndGCPFields verifies each cloud provider uses its own fact names.
//
// Given Azure or GCP marketplace facts, when parsed,
// then the provider's account, instance, and offer values are mapped.
func TestParseMapsAzureAndGCPFields(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name         string
		facts        map[string]string
		wantID       string
		wantAccount  string
		wantInstance string
		wantOffers   []string
	}{
		{
			name: "azure",
			facts: map[string]string{
				"lscpu.cpu(s)":          "4",
				"azure_instance_id":     "azure-vm",
				"azure_subscription_id": "subscription",
				"azure_offer":           "rhel-ha, rhel-sap",
			},
			wantID: "azure", wantAccount: "subscription", wantInstance: "azure-vm",
			wantOffers: []string{"rhel-ha", "rhel-sap"},
		},
		{
			name: "gcp",
			facts: map[string]string{
				"lscpu.cpu(s)":       "4",
				"gcp_instance_id":    "gcp-vm",
				"gcp_project_number": "1234",
				"gcp_license_codes":  "license-a,license-b",
			},
			wantID: "gcp", wantAccount: "1234", wantInstance: "gcp-vm",
			wantOffers: []string{"license-a", "license-b"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := extractFacts(test.facts)
			if err != nil {
				t.Fatalf("extractFacts() error = %v", err)
			}

			if got.MarketplaceID != test.wantID || got.MarketplaceAccountID != test.wantAccount ||
				got.MarketplaceInstanceID != test.wantInstance {
				t.Errorf("extractFacts() = %#v, want provider %q account %q instance %q",
					got, test.wantID, test.wantAccount, test.wantInstance)
			}

			if !equalStringsIgnoringOrder(got.MarketplaceOfferIDs, test.wantOffers) {
				t.Errorf("extractFacts() OfferIDs = %#v, want %#v in any order", got.MarketplaceOfferIDs, test.wantOffers)
			}
		})
	}
}

// TestParseLeavesAbsentMarketplaceFactsEmpty verifies non-marketplace machines need not provide cloud data.
//
// Given a valid CPU fact and no cloud facts, when parsed, then marketplace fields remain empty.
func TestParseLeavesAbsentMarketplaceFactsEmpty(t *testing.T) {
	t.Parallel()

	got, err := extractFacts(map[string]string{"unrelated_fact": "value", "lscpu.cpu(s)": "2"})
	if err != nil {
		t.Fatalf("extractFacts() error = %v", err)
	}

	if got.MarketplaceID != "" || got.MarketplaceAccountID != "" || got.MarketplaceInstanceID != "" ||
		len(got.MarketplaceOfferIDs) != 0 || got.VCPUCount == nil || *got.VCPUCount != 2 {
		t.Errorf("extractFacts() = %#v, want empty optional fields", got)
	}
}

// TestParseLeavesMissingVCPUEmpty verifies an absent CPU fact does not block profile parsing.
//
// Given a profile with no CPU count, when parsed, then the profile is returned with a zero vCPU count.
func TestParseLeavesMissingVCPUEmpty(t *testing.T) {
	t.Parallel()

	got, err := extractFacts(map[string]string{"aws_instance_id": "instance"})
	if err != nil {
		t.Fatalf("extractFacts() error = %v", err)
	}

	if got.VCPUCount != nil {
		t.Errorf("extractFacts() VCPUCount = %v, want nil", got.VCPUCount)
	}
}

// TestParseLeavesInvalidVCPUEmpty verifies invalid CPU facts do not block profile parsing.
//
// Given a missing or invalid CPU fact, when parsed, then the profile is returned with a zero vCPU count.
func TestParseLeavesInvalidVCPUEmpty(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"many", "0", "-1"} {
		got, err := extractFacts(map[string]string{"lscpu.cpu(s)": value})
		if err != nil {
			t.Errorf("extractFacts() with CPU fact %q error = %v", value, err)

			continue
		}

		if got.VCPUCount != nil {
			t.Errorf("extractFacts() with CPU fact %q VCPUCount = %v, want nil", value, got.VCPUCount)
		}
	}
}

// TestParseDetectsProviderFromAnyKnownFact verifies even a single provider-specific fact identifies a marketplace.
//
// Given any Azure-prefixed fact without the other identifiers, when parsed,
// then the marketplace is identified as Azure.
func TestParseDetectsProviderFromAnyKnownFact(t *testing.T) {
	t.Parallel()

	got, err := extractFacts(map[string]string{"azure_region": "region", "lscpu.cpu(s)": "2"})
	if err != nil {
		t.Fatalf("extractFacts() error = %v", err)
	}

	if got.MarketplaceID != "azure" {
		t.Errorf("extractFacts() MarketplaceID = %q, want azure", got.MarketplaceID)
	}
}

// equalStringsIgnoringOrder reports whether two string slices contain the same values and multiplicities.
func equalStringsIgnoringOrder(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	counts := make(map[string]int, len(left))
	for _, value := range left {
		counts[value]++
	}

	for _, value := range right {
		counts[value]--
		if counts[value] < 0 {
			return false
		}
	}

	return true
}
