package main

import (
	"testing"
)

func Test_checkIP4Health(t *testing.T) {
	type args struct {
		list string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"working RBL 1 - blacklist", args{"zen.spamhaus.org"}, true},
		{"working RBL 2 - whitelist", args{"swl.spamhaus.org"}, true},
		// {"working RBL 3", args{"b.barracudacentral.org"}, true},
		// {"working RBL 4", args{"dnsbl-0.uceprotect.net"}, true},
		// {"working RBL 5", args{"bl.spamcop.net"}, true},
		{"random domain", args{"www.example.com"}, false},
		{"non-existant domain", args{"12345.invaliddomain871253659dfd.com"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkIP4Health(tt.args.list); got != tt.want {
				t.Errorf("checkIP4Health() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkDomainHealth(t *testing.T) {
	type args struct {
		list string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"working RBL 1 - blacklist", args{"dbl.spamhaus.org"}, true},
		// {"working RBL 3", args{"b.barracudacentral.org"}, true},
		// {"working RBL 4", args{"dnsbl-0.uceprotect.net"}, true},
		// {"working RBL 5", args{"bl.spamcop.net"}, true},
		{"random domain", args{"www.example.com"}, false},
		{"non-existant domain", args{"12345.invaliddomain871253659dfd.com"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkDomainHealth(tt.args.list); got != tt.want {
				t.Errorf("checkDomainHealth() = %v, want %v", got, tt.want)
			}
		})
	}
}
