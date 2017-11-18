package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	valid "github.com/asaskevich/govalidator"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	app          = kingpin.New("dnsbl_checker", "All-in-one DNSBL checker written in Go using every publicly known DNSBL.")
	ip4Cmd       = app.Command("ip", "checks IPv4 address against DNSBLs")
	cfgWhitelist = app.Flag("whitelist", "Check whitelists instead of blacklists").Bool()
	cfgVerbose   = app.Flag("verbose", "More verbose output. Output will include misses, timeouts and failures.").Bool()
	cfgExclude   = app.Flag("exclude", "List of DNSBLs to exclude from the check. This flag can be specified multiple times.").PlaceHolder("bl.example.com").Strings()
	cfgSpeed     = app.Flag("speed", "number of checks per second between 1 (min) and 1000 (max)").Default("20").Int()
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

	// create filteredLists by removing excluded lists from allLists
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
	counterHits := make(chan bool, len(lists))
	counterMisses := make(chan bool, len(lists))
	counterTimeouts := make(chan bool, len(lists))
	counterFailures := make(chan bool, len(lists))
	counterChecks := 0

	sleep := 1000 / *cfgSpeed
	d, err := time.ParseDuration(strconv.Itoa(sleep) + "ms")
	if err != nil {
		panic(err)
	}
	ticker := time.Tick(d)

	for _, v := range lists {
		wg.Add(1)
		counterChecks++
		go func(address string, v *ListItem) {
			<-ticker
			result, err := lookupFunc(address, v)
			if err != nil {
				var out string
				if strings.HasSuffix(err.Error(), "no such host") {
					out = fmt.Sprintf("%v : MISS\n", v.Address)
					counterMisses <- true
				} else if strings.HasSuffix(err.Error(), "i/o timeout") {
					out = fmt.Sprintf("%v : TIMEOUT\n", v.Address)
					counterTimeouts <- true
				} else {
					out = fmt.Sprintf("%v : FAILURE\n", v.Address)
					counterFailures <- true
				}
				if *cfgVerbose {
					fmt.Printf("%v", out)
				}
			}
			if result {
				fmt.Printf("%v : HIT\n", v.Address)
				counterHits <- true
			}
			wg.Done()
		}(address, v)
	}

	wg.Wait()
	numListed := len(counterHits)

	fmt.Printf("------------------------------------------------\n")
	fmt.Printf("Result: %v checks performed. %v hits, %v misses, %v timeouts, %v failures\n", counterChecks, len(counterHits), len(counterMisses), len(counterTimeouts), len(counterFailures))

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
