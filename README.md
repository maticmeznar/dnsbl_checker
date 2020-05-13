# dnsbl_checker
All-in-one DNSBL checker written in Go using every publicly known DNSBL.

DNSBL is also known as RBL or DNS blacklist.

## Features
- Fast. All lists are checked simultaneously. It takes about 10 seconds.
- Cross-platform. Works on Windows, macOS and Linux.
- No dependencies. Everything you need to run `dnsbl_checker` is in a single file.
- Nagios/Icing/Sensu compatible. `dnsbl_checker` exits with the appropriate exit code.
- Complete. `dnsbl_checker` can check IPv4 addresses and domains. All against blacklists and whitelists.
- Flexible. You can exclude one or more DNSBLs from the check, or only check against a select few.

## Other
- IPv6 is not supported because it's mostly useless in DNSBL context. Best solution is to not bind your SMTP server to an IPv6 address, so you cannot receive any email from IPv6 sources.

This checker uses DNSBL list from http://multirbl.valli.org/list/. HTML source of the table is used to create a CSV list using http://www.convertcsv.com/html-table-to-csv.htm or https://conversiontools.io/convert_html_to_csv/.

## Additional resources
- https://tools.ietf.org/html/rfc5782#page-7