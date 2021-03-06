# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]

- Vendoring using Go modules
- Improved concurrency
- DNSBL's health is checked before use

## [0.2.1] - 2019-06-09

### Changed

- Updated DNSBL list

## [0.2] - 2017-11-19

### Added

- Rate of checks per second can now be set with `--speed` flag.
- More verbose output can now be enabled with `--verbose` flag.

### Changed

- Command 'ip4' was renamed to 'ip'.
- Some DNSBL lists were disabled because they weren't working.

## [0.1] - 2017-11-17

### Added

- DNSBL IPv4 queries
- DNSBL domain queries
- Ability to exclude one or more lists from checks
- Nagios compatible exit codes
- List of 328 DNSBLs
