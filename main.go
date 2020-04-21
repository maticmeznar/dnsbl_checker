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
	ip4Cmd       = app.Command("ip", "checks IPv4 address against DNSBLs")
	cfgWhitelist = app.Flag("whitelist", "Check whitelists instead of blacklists").Bool()
	cfgVerbose   = app.Flag("verbose", "More verbose output. Output will include misses, timeouts and failures.").Bool()
	cfgExclude   = app.Flag("exclude", "List of DNSBLs to exclude from the check. This flag can be specified multiple times.").PlaceHolder("bl.example.com").Strings()
	cfgThreads   = app.Flag("threads", "number of concurrent checks between 1 (min) and 1000 (max)").Default("10").Int()
	cfgIP4       = ip4Cmd.Arg("ip", "IP address to check").Required().String()
	// ip6Cmd       = app.Command("ip6", "checks IPv6 address against DNSBLs")
	// cfgIP6       = ip6Cmd.Arg("ip", "IP address to check").Required().String()
	domainCmd          = app.Command("domain", "checks a domain against DNSBLs")
	cfgDomain          = domainCmd.Arg("domain", "domain name to check").Required().String()
	version            = "0.2"
	ErrWrongResponse   = fmt.Errorf("RBL returned a response outside of 127.0.0.0/8 subnet")
	ErrRBLPositiveFail = fmt.Errorf("RBL failed positive check")
	ErrRBLNegativeFail = fmt.Errorf("RBL failed negative check")
	ErrRBLFail         = fmt.Errorf("RBL failed both checks")
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

type workUnit struct {
	// address is the hostname used for checking
	address string
	// listItem is the ListItem
	listItem *ListItem

	counterHits     chan bool
	counterMisses   chan bool
	counterTimeouts chan bool
	counterFailures chan bool
	lookupFunc      func(string, *ListItem) (bool, error)
}

func main() {
	app.Version(version)

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
	// check RBL health before using it
	if err := checkIP4Health(list.Address); err != nil {
		return false, err
	}

	stringyIP := strings.Split(ip, ".")
	addr := stringyIP[3] + "." + stringyIP[2] + "." + stringyIP[1] + "." + stringyIP[0] + "." + list.Address

	ips, err := net.LookupIP(addr)
	if err != nil {
		return false, err
	}

	if len(ips) > 0 {
		_, subNet, _ := net.ParseCIDR("127.0.0.0/8")
		if !subNet.Contains(ips[0]) {
			return false, ErrWrongResponse
		}
		return true, nil
	}

	return false, nil
}

// lookupDomain returns true if `domain` is listed, false otherwise
func lookupDomain(domain string, list *ListItem) (bool, error) {
	// check RBL health before using it
	if !checkDomainHealth(list.Address) {
		fmt.Printf("%v : ERROR\n", list.Address)
		return false, fmt.Errorf("RBL is not working properly")
	}

	addrs, err := net.LookupHost(domain + "." + list.Address)
	if err != nil {
		return false, err
	}

	if len(addrs) > 0 {
		return true, nil
	}

	return false, nil
}

func worker(wg *sync.WaitGroup, ch chan *workUnit, done chan bool) {
	wg.Add(1)
	for {
		select {
		case <-done:
			wg.Done()
			return
		case wu := <-ch:
			result, err := wu.lookupFunc(wu.address, wu.listItem)
			if err != nil {
				var out string
				if strings.HasSuffix(err.Error(), "no such host") {
					out = fmt.Sprintf("%v : MISS\n", wu.listItem.Address)
					wu.counterMisses <- true
				} else if strings.HasSuffix(err.Error(), "i/o timeout") {
					out = fmt.Sprintf("%v : TIMEOUT\n", wu.listItem.Address)
					wu.counterTimeouts <- true
				} else {
					out = fmt.Sprintf("%v : FAILURE: %v\n", wu.listItem.Address, err)
					wu.counterFailures <- true
				}
				if *cfgVerbose {
					fmt.Printf("%v", out)
				}
			}
			if result {
				fmt.Printf("%v : HIT\n", wu.listItem.Address)
				wu.counterHits <- true
			}

		}
	}

}

func runChecks(address string, lists []*ListItem, lookupFunc func(string, *ListItem) (bool, error)) {
	wg := &sync.WaitGroup{}
	counterHits := make(chan bool, len(lists))
	counterMisses := make(chan bool, len(lists))
	counterTimeouts := make(chan bool, len(lists))
	counterFailures := make(chan bool, len(lists))
	counterChecks := 0
	workChan := make(chan *workUnit)
	workDone := make(chan bool)

	for i := 1; i <= *cfgThreads; i++ {
		go worker(wg, workChan, workDone)
	}

	for _, listItem := range lists {
		counterChecks++

		wu := &workUnit{
			address:         address,
			listItem:        listItem,
			counterFailures: counterFailures,
			counterHits:     counterHits,
			counterMisses:   counterMisses,
			counterTimeouts: counterTimeouts,
			lookupFunc:      lookupFunc,
		}

		workChan <- wu
	}

	close(workDone)
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

// checkIP4Health returns true if `list` is healthy. Returns false otherwise.
// Test specification: https://tools.ietf.org/html/rfc5782#page-7
func checkIP4Health(list string) error {
	testNegative := func(list string) bool {
		ips, err := net.LookupHost("1.0.0.127" + "." + list)
		if len(ips) == 0 && strings.HasSuffix(err.Error(), "no such host") {
			return true
		}
		return false
	}

	testPositive := func(list string) bool {
		ips, err := net.LookupHost("2.0.0.127" + "." + list)
		if len(ips) > 0 && err == nil {
			return true
		}
		return false
	}

	negResult := testNegative(list)
	posResult := testPositive(list)

	if !negResult && !posResult {
		return ErrRBLFail
	} else if negResult && !posResult {
		return ErrRBLPositiveFail
	} else if !negResult && posResult {
		return ErrRBLNegativeFail
	}

	return nil
}

// checkDomainHealth returns true if `list` is healthy. Returns false otherwise.
// Test specification: https://tools.ietf.org/html/rfc5782#page-7
func checkDomainHealth(list string) bool {
	testNegative := func(list string) bool {
		ips, err := net.LookupHost("INVALID" + "." + list)
		if len(ips) == 0 && strings.HasSuffix(err.Error(), "no such host") {
			return true
		}
		return false
	}

	testPositive := func(list string) bool {
		ips, err := net.LookupHost("TEST" + "." + list)
		if len(ips) > 0 && err == nil {
			return true
		}
		return false
	}

	if testNegative(list) && testPositive(list) {
		return true
	}

	return false
}
