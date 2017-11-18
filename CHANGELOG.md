# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Changed
- Renamed 'ip4' command to 'ip'
- Rate of checks per second can now be set with `--speed` flag.
- More verbose output can now be enabled with `--verbose` flag.

## [0.1] - 2017-11-17
### Added
- DNSBL IPv4 queries
- DNSBL domain queries
- Ability to exclude one or more lists from checks
- Nagios compatible exit codes
- List of 328 DNSBLs
