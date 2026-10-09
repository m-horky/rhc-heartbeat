package config

import "testing"

// TestValidateBypassRuleCoversDocumentedForms verifies accepted and rejected no-proxy rules.
//
// Given supported host, IP, CIDR, port, and wildcard rules plus malformed entries, when validation runs,
// then only the documented rule forms are accepted.
func TestValidateBypassRuleCoversDocumentedForms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		rule string
		want bool
	}{
		{"example.com", true},
		{"*.example.com", true},
		{"*", true},
		{"notexample.com", true},
		{"192.0.2.5", true},
		{"10.0.0.0/8", true},
		{"2001:db8::/32", true},
		{"example.com:443", true},
		{"[2001:db8::1]:443", true},
		{"bad**.example.com", false},
		{"example.com:99999", false},
		{"[2001:db8::1]:bad", false},
		{"example..com", false},
	}
	for _, test := range tests {
		t.Run(test.rule, func(t *testing.T) {
			if got := validateBypassRule(test.rule); got != test.want {
				t.Errorf("validateBypassRule(%q) = %v, want %v", test.rule, got, test.want)
			}
		})
	}
}
