package auth

import (
	"net/netip"
	"testing"
)

// Login alerts the user when both onSeenNetwork and inSeenCountry return false, so a wrong match means missed or spurious alerts.
func TestOnSeenNetwork(t *testing.T) {
	cases := []struct {
		ip   string
		seen []string
		want bool
	}{
		{"203.0.113.200", []string{"203.0.113.7"}, true},           // same IPv4 /24
		{"203.0.114.7", []string{"203.0.113.7"}, false},            // next /24 over
		{"203.0.113.9", []string{"::ffff:203.0.113.7"}, true},      // IPv4-mapped IPv6 counts as IPv4
		{"2001:db8:1:bb::2", []string{"2001:db8:1:aa::1"}, true},   // same IPv6 /48
		{"2001:db8:2::1", []string{"2001:db8:1::1"}, false},        // different /48
		{"203.0.113.7", []string{"2001:db8::1", "bad", ""}, false}, // no IPv4 history, junk rows skipped
		{"", []string{"203.0.113.7"}, false},                       // missing client IP
		{"203.0.113.7", nil, false},                                // no history
	}
	for _, c := range cases {
		if got := onSeenNetwork(c.ip, c.seen); got != c.want {
			t.Errorf("onSeenNetwork(%q, %q) = %v, want %v", c.ip, c.seen, got, c.want)
		}
	}
}

func TestInSeenCountry(t *testing.T) {
	countries := map[string]string{"203.0.113.7": "AU", "198.51.100.9": "AU", "192.0.2.1": "NZ"}
	country := func(a netip.Addr) string { return countries[a.String()] }

	cases := []struct {
		ip   string
		seen []string
		want bool
	}{
		{"198.51.100.9", []string{"203.0.113.7"}, true},           // different network, same country
		{"192.0.2.1", []string{"203.0.113.7"}, false},             // different country
		{"::ffff:198.51.100.9", []string{"203.0.113.7"}, true},    // IPv4-mapped IPv6 looked up as IPv4
		{"10.0.0.1", []string{"10.0.0.2"}, false},                 // both unknown: "" must not match ""
		{"198.51.100.9", []string{"bad", "", "192.0.2.1"}, false}, // junk rows skipped
		{"", []string{"203.0.113.7"}, false},                      // missing client IP
	}
	for _, c := range cases {
		if got := inSeenCountry(c.ip, c.seen, country); got != c.want {
			t.Errorf("inSeenCountry(%q, %q) = %v, want %v", c.ip, c.seen, got, c.want)
		}
	}
}
