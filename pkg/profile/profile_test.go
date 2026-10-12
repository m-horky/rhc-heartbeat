package profile

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestNewCPUProfile reads logical CPU count.
//
// Given a lscpu.cpu(s) fact, when a CPU profile is created, then its count matches the fact.
func TestNewCPUProfile(t *testing.T) {
	t.Parallel()

	got := newCPUProfile(map[string]string{"lscpu.cpu(s)": "12"})
	if got.VCPUCount == nil || *got.VCPUCount != uint64(12) {
		t.Errorf("newCPUProfile() VCPUCount = %v, want 12", got.VCPUCount)
	}
}

// TestNewCPUProfileError validates absence of the fact is handled.
//
// Given a facts set without lscpu.cpu(s), when a CPU profile is created,
// then a cpuProfile with nil vCPU value is created.
func TestNewCPUProfileError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   map[string]string
	}{
		{
			name: "no facts",
			in:   map[string]string{},
		},
		{
			name: "no lscpu.cpu(s)",
			in:   map[string]string{"cpu.cpu(s)": "12"},
		},
		{
			name: "malformed lscpu.cpu(s)",
			in:   map[string]string{"lscpu.cpu(s)": "not available"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			want := cpuProfile{VCPUCount: nil}

			got := newCPUProfile(test.in)
			if !cmp.Equal(want, got) {
				t.Errorf("newCPUProfile() got = %v, want %v", got, want)
			}
		})
	}
}

// TestNewMarketplaceProfile reads public cloud profile.
//
// Given a set of sub-man facts, when a marketplace profile is created,
// then the appropriate facts are merged.
func TestNewMarketplaceProfile(t *testing.T) { //nolint:funlen
	t.Parallel()

	tests := []struct {
		name string
		in   map[string]string
		want marketplaceProfile
	}{
		{
			name: "not marketplace",
			in:   map[string]string{},
			want: marketplaceProfile{},
		},
		{
			name: "aws without offers",
			in: map[string]string{
				"aws_instance_id": "aws-instance",
				"aws_account_id":  "aws-account",
			},
			want: marketplaceProfile{
				ID:       new("aws"),
				Account:  new("aws-account"),
				Instance: new("aws-instance"),
			},
		},
		{
			name: "aws with billing products",
			in: map[string]string{
				"aws_instance_id":      "aws-instance",
				"aws_account_id":       "aws-account",
				"aws_billing_products": "billing-product",
			},
			want: marketplaceProfile{
				ID:       new("aws"),
				Account:  new("aws-account"),
				Instance: new("aws-instance"),
				Offers:   new([]string{"billing-product"}),
			},
		},
		{
			name: "aws with product codes",
			in: map[string]string{
				"aws_instance_id":               "aws-instance",
				"aws_account_id":                "aws-account",
				"aws_marketplace_product_codes": "product-code",
			},
			want: marketplaceProfile{
				ID:       new("aws"),
				Account:  new("aws-account"),
				Instance: new("aws-instance"),
				Offers:   new([]string{"product-code"}),
			},
		},
		{
			name: "azure without offers",
			in: map[string]string{
				"azure_instance_id":     "azure-instance",
				"azure_subscription_id": "azure-account",
			},
			want: marketplaceProfile{
				ID:       new("azure"),
				Account:  new("azure-account"),
				Instance: new("azure-instance"),
			},
		},
		{
			name: "azure with offers",
			in: map[string]string{
				"azure_instance_id":     "azure-instance",
				"azure_subscription_id": "azure-account",
				"azure_offer":           "offer",
			},
			want: marketplaceProfile{
				ID:       new("azure"),
				Account:  new("azure-account"),
				Instance: new("azure-instance"),
				Offers:   new([]string{"offer"}),
			},
		},
		{
			name: "gcp without codes",
			in: map[string]string{
				"gcp_instance_id":    "gcp-instance",
				"gcp_project_number": "gcp-account",
			},
			want: marketplaceProfile{
				ID:       new("gcp"),
				Account:  new("gcp-account"),
				Instance: new("gcp-instance"),
			},
		},
		{
			name: "gcp with codes",
			in: map[string]string{
				"gcp_instance_id":    "gcp-instance",
				"gcp_project_number": "gcp-account",
				"gcp_license_codes":  "license-code",
			},
			want: marketplaceProfile{
				ID:       new("gcp"),
				Account:  new("gcp-account"),
				Instance: new("gcp-instance"),
				Offers:   new([]string{"license-code"}),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := newMarketplaceProfile(test.in)
			if diff := cmp.Diff(got, test.want); diff != "" {
				t.Errorf("newMarketplaceProfile() (-got +want):\n%s", diff)
			}
		})
	}
}
