package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	valid "github.com/asaskevich/govalidator"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	app          = kingpin.New("dnsbl_checker", "All-in-one DNSBL checker written in Go using every publicly known DNSBL.")
	ip4Cmd       = app.Command("ip4", "checks IPv4 address against DNSBLs")
	cfgWhitelist = app.Flag("whitelist", "Check whitelists instead of blacklists").Bool()
	cfgExclude   = app.Flag("exclude", "List of DNSBLs to exclude from the check. This flag can be specified multiple times.").PlaceHolder("bl.example.com").Strings()
	cfgIP4       = ip4Cmd.Arg("ip", "IP address to check").Required().String()
	// ip6Cmd       = app.Command("ip6", "checks IPv6 address against DNSBLs")
	// cfgIP6       = ip6Cmd.Arg("ip", "IP address to check").Required().String()
	domainCmd = app.Command("domain", "checks a domain against DNSBLs")
	cfgDomain = domainCmd.Arg("domain", "domain name to check").Required().String()
	version   = "0.1"
	author    = "Matic Mežnar <matic@meznar.si>"
)

// ListItem is a struct with list details
type ListItem struct {
	// Name is the name of the list
	Name string
	// Address is the hostname used for checking
	Address string
	// IP4 is true if this list is used for checking IP4 addresses
	IP4 bool
	// IP6 is true if this list is used for checking IP4 addresses
	IP6 bool
	// Domain is true if this list is used for checking domains
	Domain bool
	// Blacklist is true if this list is a blacklist
	Blacklist bool
	// Whitelist is true if this list is a whitelist
	Whitelist bool
}

func main() {
	app.Version(version)
	app.Author(author)

	ks := kingpin.MustParse(app.Parse(os.Args[1:]))
	allLists := parseCVS()
	filteredLists := []*ListItem{}

	// TODO: remove excluded lists
	if len(*cfgExclude) >= 1 {
		for _, vAll := range allLists {
			if isStringInSlice(vAll.Address, *cfgExclude) {
				continue
			}
			filteredLists = append(filteredLists, vAll)
		}
	} else {
		filteredLists = allLists
	}

	switch ks {
	case ip4Cmd.FullCommand():
		if !valid.IsIPv4(*cfgIP4) {
			app.FatalUsage("You have not supplied a valid IP4 address.")
		}
		CheckIP4(*cfgWhitelist, *cfgIP4, filteredLists)

	// case ip6Cmd.FullCommand():
	// 	if !valid.IsIPv6(*cfgIP6) {
	// 		app.FatalUsage("You have not supplied a valid IP6 address.")
	// 	}
	// CheckIP6(*cfgWhitelist, *cfgIP6, filteredLists)

	case domainCmd.FullCommand():
		if !valid.IsDNSName(*cfgDomain) {
			app.FatalUsage("You have not supplied a valid domain name.")
		}
		CheckDomain(*cfgWhitelist, *cfgDomain, filteredLists)
	}

}

// CheckIP4 .
func CheckIP4(whitelist bool, ip string, allLists []*ListItem) {
	lists := []*ListItem{}
	for _, v := range allLists {
		if v.IP4 && v.Whitelist == whitelist {
			lists = append(lists, v)
		}
	}

	runChecks(ip, lists, lookupIP4)
}

// CheckDomain .
func CheckDomain(whitelist bool, domain string, allLists []*ListItem) {
	lists := []*ListItem{}
	for _, v := range allLists {
		if v.Domain && v.Whitelist == whitelist {
			lists = append(lists, v)
		}
	}

	runChecks(domain, lists, lookupDomain)
}

// lookupIP4 returns true if `ip` is listed, false otherwise
func lookupIP4(ip string, list *ListItem) (bool, error) {
	stringyIP := strings.Split(ip, ".")
	addr := stringyIP[3] + "." + stringyIP[2] + "." + stringyIP[1] + "." + stringyIP[0] + "." + list.Address

	ips, err := net.LookupIP(addr)
	if err != nil {
		return false, err
	}

	if len(ips) > 0 {
		return true, nil
	}

	return false, nil
}

// lookupDomain returns true if `domain` is listed, false otherwise
func lookupDomain(domain string, list *ListItem) (bool, error) {
	addrs, err := net.LookupHost(domain + "." + list.Address)
	if err != nil {
		return false, err
	}

	if len(addrs) > 0 {
		return true, nil
	}

	return false, nil
}

func runChecks(address string, lists []*ListItem, lookupFunc func(string, *ListItem) (bool, error)) {
	wg := sync.WaitGroup{}
	counterBad := make(chan bool, len(lists))
	counterChecks := 0

	for _, v := range lists {
		wg.Add(1)
		counterChecks++
		go func(address string, v *ListItem) {
			result, _ := lookupFunc(address, v)
			if result {
				fmt.Printf("%v is listed in %v\n", address, v.Address)
				// spew.Dump(v)
				counterBad <- true
			}
			wg.Done()
		}(address, v)
	}

	wg.Wait()
	numListed := len(counterBad)

	fmt.Printf("------------------------------------------------\n")
	fmt.Printf("Result: %v checks performed. %v is listed %v times.\n", counterChecks, address, numListed)

	if numListed > 0 && !*cfgWhitelist {
		os.Exit(2)
	}

	os.Exit(0)
}

// isStringInSlice returns true if `needle` is in `haystrack`
func isStringInSlice(needle string, haystrack []string) bool {
	for _, v := range haystrack {
		if needle == v {
			return true
		}
	}

	return false
}
