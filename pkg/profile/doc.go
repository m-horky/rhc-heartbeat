// Package profile derives system profile data, including cloud marketplace
// metadata and vCPU count, from subscription-manager facts.
//
// Get reads facts from the local subscription-manager command and derives a
// profile.
//
// To read marketplace metadata and the vCPU count:
//
//	systemProfile, err := profile.Get()
//	if err != nil {
//		return err
//	}
//	marketplace := systemProfile.MarketplaceID
//	var vcpuCount uint64
//	if systemProfile.VCPUCount != nil {
//		vcpuCount = *systemProfile.VCPUCount
//	}
//	_, _ = marketplace, vcpuCount
package profile
