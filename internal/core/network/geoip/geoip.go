// Package geoip provides offline IP-to-country geolocation mapping.
package geoip

import (
	"encoding/binary"
	"net/netip"
	"sort"
	"strings"
)

// Location describes the geographical origin or ownership of an IP address.
type Location struct {
	CountryCode string  // ISO 3166-1 alpha-2 or internal code ("DE", "US", "IR", "FI", "NL", "LOCAL")
	CountryName string  // Full name ("Germany", "United States", "Iran", "Finland", etc.)
	Flag        string  // Flag emoji or symbol ("🇩🇪", "🇺🇸", "🇮🇷", "🇫🇮", "🇳🇱", "🏠")
	IsLocal     bool    // True for loopback, RFC1918, link-local
	Latitude    float64 // Geographical latitude (-90 to +90)
	Longitude   float64 // Geographical longitude (-180 to +180)
}

type ipRange struct {
	start uint32
	end   uint32
	code  string
	name  string
	flag  string
}

var ranges []ipRange

func ip4(a, b, c, d byte) uint32 {
	return (uint32(a) << 24) | (uint32(b) << 16) | (uint32(c) << 8) | uint32(d)
}

func cidr(a, b, c, d byte, mask int) (uint32, uint32) {
	ip := ip4(a, b, c, d)
	m := uint32(0)
	if mask > 0 {
		m = ^uint32(0) << (32 - mask)
	}
	start := ip & m
	end := start | ^m
	return start, end
}

func addCIDR(a, b, c, d byte, mask int, code, name, flag string) {
	start, end := cidr(a, b, c, d, mask)
	ranges = append(ranges, ipRange{start: start, end: end, code: code, name: name, flag: flag})
}

func addRange(sA, sB, sC, sD, eA, eB, eC, eD byte, code, name, flag string) {
	ranges = append(ranges, ipRange{
		start: ip4(sA, sB, sC, sD),
		end:   ip4(eA, eB, eC, eD),
		code:  code,
		name:  name,
		flag:  flag,
	})
}

func init() {
	// --- High-priority specific datacenter, cloud, and country subnets ---

	// Finland (Hetzner Helsinki datacenters & Finnish telecom)
	addRange(65, 108, 0, 0, 65, 109, 255, 255, "FI", "Finland", "🇫🇮")
	addRange(95, 216, 0, 0, 95, 217, 255, 255, "FI", "Finland", "🇫🇮")
	addRange(135, 181, 0, 0, 135, 181, 255, 255, "FI", "Finland", "🇫🇮")

	// Germany (Hetzner Falkenstein/Nuremberg, Deutsche Telekom, etc.)
	addRange(148, 251, 0, 0, 148, 251, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(116, 202, 0, 0, 116, 203, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(159, 69, 0, 0, 159, 69, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(168, 119, 0, 0, 168, 119, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(195, 201, 0, 0, 195, 201, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(138, 201, 0, 0, 138, 201, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(78, 46, 0, 0, 78, 47, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(88, 99, 0, 0, 88, 99, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(85, 10, 192, 0, 85, 10, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(46, 4, 0, 0, 46, 4, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(144, 76, 0, 0, 144, 76, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(5, 9, 0, 0, 5, 9, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(94, 130, 0, 0, 94, 130, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(188, 114, 96, 0, 188, 114, 111, 255, "DE", "Germany (Cloudflare)", "🇩🇪")

	// Iran (major ISP / Datacenter ranges)
	addRange(2, 144, 0, 0, 2, 147, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(5, 106, 0, 0, 5, 107, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(5, 112, 0, 0, 5, 127, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(5, 160, 0, 0, 5, 161, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(5, 200, 0, 0, 5, 203, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(5, 208, 0, 0, 5, 215, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(5, 220, 0, 0, 5, 223, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(5, 232, 0, 0, 5, 239, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(31, 2, 0, 0, 31, 2, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(31, 7, 0, 0, 31, 7, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(31, 24, 0, 0, 31, 31, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(37, 98, 0, 0, 37, 115, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(37, 152, 0, 0, 37, 153, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(37, 254, 0, 0, 37, 255, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(78, 38, 0, 0, 78, 39, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(79, 127, 0, 0, 79, 127, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(79, 175, 0, 0, 79, 175, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(80, 191, 0, 0, 80, 191, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(85, 185, 0, 0, 85, 185, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(89, 165, 0, 0, 89, 165, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(91, 98, 0, 0, 91, 99, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(94, 182, 0, 0, 94, 184, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(151, 232, 0, 0, 151, 247, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(178, 131, 0, 0, 178, 131, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(185, 8, 0, 0, 185, 8, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(185, 55, 0, 0, 185, 55, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(185, 143, 0, 0, 185, 143, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(188, 158, 0, 0, 188, 159, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(188, 210, 0, 0, 188, 213, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(188, 253, 0, 0, 188, 253, 255, 255, "IR", "Iran", "🇮🇷")
	addRange(217, 218, 0, 0, 217, 219, 255, 255, "IR", "Iran", "🇮🇷")

	// Netherlands (Telegram, AMS-IX, Leaseweb)
	addRange(149, 154, 160, 0, 149, 154, 175, 255, "NL", "Netherlands", "🇳🇱")
	addRange(91, 108, 4, 0, 91, 108, 7, 255, "NL", "Netherlands", "🇳🇱")
	addRange(91, 108, 56, 0, 91, 108, 59, 255, "NL", "Netherlands", "🇳🇱")
	addRange(145, 0, 0, 0, 145, 255, 255, 255, "NL", "Netherlands", "🇳🇱")

	// United States (Major Tech Backends & CDNs)
	addRange(142, 54, 0, 0, 142, 54, 255, 255, "US", "United States", "🇺🇸")
	addRange(142, 250, 0, 0, 142, 251, 255, 255, "US", "United States", "🇺🇸") // Google
	addRange(172, 217, 0, 0, 172, 217, 255, 255, "US", "United States", "🇺🇸") // Google
	addRange(8, 8, 4, 0, 8, 8, 8, 255, "US", "United States", "🇺🇸")           // Google DNS
	addRange(1, 1, 1, 0, 1, 1, 1, 255, "US", "United States", "🇺🇸")           // Cloudflare DNS
	addRange(104, 16, 0, 0, 104, 31, 255, 255, "US", "United States", "🇺🇸")   // Cloudflare
	addRange(172, 64, 0, 0, 172, 71, 255, 255, "US", "United States", "🇺🇸")   // Cloudflare
	addRange(140, 82, 112, 0, 140, 82, 127, 255, "US", "United States", "🇺🇸") // GitHub
	addRange(185, 199, 108, 0, 185, 199, 111, 255, "US", "United States", "🇺🇸") // GitHub
	addRange(13, 64, 0, 0, 13, 107, 255, 255, "US", "United States", "🇺🇸")    // Microsoft
	addRange(20, 0, 0, 0, 20, 255, 255, 255, "US", "United States", "🇺🇸")     // Azure
	addRange(52, 0, 0, 0, 54, 255, 255, 255, "US", "United States", "🇺🇸")     // AWS
	addRange(3, 0, 0, 0, 3, 255, 255, 255, "US", "United States", "🇺🇸")       // AWS
	addRange(34, 0, 0, 0, 35, 255, 255, 255, "US", "United States", "🇺🇸")     // GCP
	addRange(157, 240, 0, 0, 157, 240, 255, 255, "US", "United States", "🇺🇸") // Meta
	addRange(18, 0, 0, 0, 18, 255, 255, 255, "US", "United States", "🇺🇸")     // AWS

	// France (OVH, Scaleway)
	addRange(51, 0, 0, 0, 51, 255, 255, 255, "FR", "France", "🇫🇷")
	addRange(164, 132, 0, 0, 164, 132, 255, 255, "FR", "France", "🇫🇷")
	addRange(178, 32, 0, 0, 178, 33, 255, 255, "FR", "France", "🇫🇷")
	addRange(54, 36, 0, 0, 54, 38, 255, 255, "FR", "France", "🇫🇷")

	// United Kingdom
	addRange(51, 140, 0, 0, 51, 145, 255, 255, "GB", "United Kingdom", "🇬🇧")
	addRange(25, 0, 0, 0, 25, 255, 255, 255, "GB", "United Kingdom", "🇬🇧")
	addRange(81, 130, 0, 0, 81, 149, 255, 255, "GB", "United Kingdom", "🇬🇧")

	// Switzerland
	addRange(185, 183, 0, 0, 185, 183, 255, 255, "CH", "Switzerland", "🇨🇭")
	addRange(195, 176, 0, 0, 195, 176, 255, 255, "CH", "Switzerland", "🇨🇭")

	// Sweden
	addRange(193, 180, 0, 0, 193, 181, 255, 255, "SE", "Sweden", "🇸🇪")
	addRange(185, 157, 0, 0, 185, 157, 255, 255, "SE", "Sweden", "🇸🇪")

	// Singapore
	addRange(103, 200, 0, 0, 103, 200, 255, 255, "SG", "Singapore", "🇸🇬")
	addRange(128, 199, 0, 0, 128, 199, 255, 255, "SG", "Singapore", "🇸🇬")

	// Japan
	addRange(133, 0, 0, 0, 133, 255, 255, 255, "JP", "Japan", "🇯🇵")
	addRange(150, 0, 0, 0, 150, 255, 255, 255, "JP", "Japan", "🇯🇵")

	// Canada
	addRange(142, 222, 0, 0, 142, 223, 255, 255, "CA", "Canada", "🇨🇦")

	// --- Global RIR Allocation Blocks (fallback regional routing) ---

	// North America (ARIN)
	addCIDR(4, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(8, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(12, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(17, 0, 0, 0, 8, "US", "United States", "🇺🇸") // Apple
	addCIDR(23, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(24, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(32, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(38, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(40, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(44, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(50, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(63, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(64, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(66, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(67, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(68, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(69, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(70, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(71, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(72, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(73, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(74, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(75, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(76, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(96, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(97, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(98, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(99, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(107, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(108, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(129, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(130, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(131, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(132, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(134, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(136, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(137, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(152, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(162, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(173, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(174, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(184, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(198, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(199, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(204, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(205, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(206, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(207, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(208, 0, 0, 0, 8, "US", "United States", "🇺🇸")
	addCIDR(209, 0, 0, 0, 8, "US", "United States", "🇺🇸")

	// Europe (RIPE NCC) - Mapped to concrete countries
	// 2.0.0.0/8
	addRange(2, 0, 0, 0, 2, 15, 255, 255, "FR", "France (Orange)", "🇫🇷")
	addRange(2, 16, 0, 0, 2, 31, 255, 255, "GB", "United Kingdom", "🇬🇧")
	addRange(2, 32, 0, 0, 2, 47, 255, 255, "IT", "Italy", "🇮🇹")
	addRange(2, 48, 0, 0, 2, 63, 255, 255, "ES", "Spain", "🇪🇸")
	addRange(2, 64, 0, 0, 2, 79, 255, 255, "PL", "Poland", "🇵🇱")
	addRange(2, 80, 0, 0, 2, 95, 255, 255, "RU", "Russia", "🇷🇺")
	addRange(2, 96, 0, 0, 2, 127, 255, 255, "GB", "United Kingdom", "🇬🇧")
	addRange(2, 128, 0, 0, 2, 159, 255, 255, "FR", "France", "🇫🇷")
	addRange(2, 160, 0, 0, 2, 223, 255, 255, "DE", "Germany (Deutsche Telekom)", "🇩🇪")
	addRange(2, 224, 0, 0, 2, 255, 255, 255, "DE", "Germany", "🇩🇪")

	// 31.0.0.0/8
	addRange(31, 0, 0, 0, 31, 15, 255, 255, "GB", "United Kingdom", "🇬🇧")
	addRange(31, 16, 0, 0, 31, 31, 255, 255, "RU", "Russia", "🇷🇺")
	addRange(31, 32, 0, 0, 31, 63, 255, 255, "ES", "Spain", "🇪🇸")
	addRange(31, 64, 0, 0, 31, 127, 255, 255, "PL", "Poland", "🇵🇱")
	addRange(31, 128, 0, 0, 31, 191, 255, 255, "RU", "Russia", "🇷🇺")
	addRange(31, 192, 0, 0, 31, 255, 255, 255, "DE", "Germany", "🇩🇪")

	// 46.0.0.0/8
	addRange(46, 0, 0, 0, 46, 63, 255, 255, "RU", "Russia", "🇷🇺")
	addRange(46, 64, 0, 0, 46, 127, 255, 255, "GB", "United Kingdom", "🇬🇧")
	addRange(46, 128, 0, 0, 46, 191, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(46, 192, 0, 0, 46, 255, 255, 255, "IT", "Italy", "🇮🇹")

	// 62.0.0.0/8
	addRange(62, 0, 0, 0, 62, 63, 255, 255, "GB", "United Kingdom", "🇬🇧")
	addRange(62, 64, 0, 0, 62, 127, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(62, 128, 0, 0, 62, 191, 255, 255, "FR", "France", "🇫🇷")
	addRange(62, 192, 0, 0, 62, 255, 255, 255, "IT", "Italy", "🇮🇹")

	// 77.0.0.0/8
	addRange(77, 0, 0, 0, 77, 63, 255, 255, "RU", "Russia", "🇷🇺")
	addRange(77, 64, 0, 0, 77, 127, 255, 255, "GB", "United Kingdom", "🇬🇧")
	addRange(77, 128, 0, 0, 77, 191, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(77, 192, 0, 0, 77, 255, 255, 255, "FR", "France", "🇫🇷")

	// 80.0.0.0/8
	addRange(80, 0, 0, 0, 80, 63, 255, 255, "GB", "United Kingdom", "🇬🇧")
	addRange(80, 64, 0, 0, 80, 127, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(80, 128, 0, 0, 80, 191, 255, 255, "FR", "France", "🇫🇷")
	addRange(80, 192, 0, 0, 80, 255, 255, 255, "IT", "Italy", "🇮🇹")

	// 81.0.0.0/8
	addRange(81, 0, 0, 0, 81, 63, 255, 255, "GB", "United Kingdom", "🇬🇧")
	addRange(81, 64, 0, 0, 81, 127, 255, 255, "FR", "France", "🇫🇷")
	addRange(81, 128, 0, 0, 81, 191, 255, 255, "IT", "Italy", "🇮🇹")
	addRange(81, 192, 0, 0, 81, 255, 255, 255, "DE", "Germany", "🇩🇪")

	// 82.0.0.0/8
	addRange(82, 0, 0, 0, 82, 63, 255, 255, "GB", "United Kingdom", "🇬🇧")
	addRange(82, 64, 0, 0, 82, 127, 255, 255, "FR", "France", "🇫🇷")
	addRange(82, 128, 0, 0, 82, 191, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(82, 192, 0, 0, 82, 255, 255, 255, "ES", "Spain", "🇪🇸")

	// 83.0.0.0/8
	addRange(83, 0, 0, 0, 83, 63, 255, 255, "ES", "Spain", "🇪🇸")
	addRange(83, 64, 0, 0, 83, 127, 255, 255, "PL", "Poland", "🇵🇱")
	addRange(83, 128, 0, 0, 83, 191, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(83, 192, 0, 0, 83, 255, 255, 255, "GB", "United Kingdom", "🇬🇧")

	// 84.0.0.0/8
	addRange(84, 0, 0, 0, 84, 63, 255, 255, "GB", "United Kingdom", "🇬🇧")
	addRange(84, 64, 0, 0, 84, 127, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(84, 128, 0, 0, 84, 191, 255, 255, "FR", "France", "🇫🇷")
	addRange(84, 192, 0, 0, 84, 255, 255, 255, "NL", "Netherlands", "🇳🇱")

	// 85.0.0.0/8
	addRange(85, 0, 0, 0, 85, 63, 255, 255, "ES", "Spain", "🇪🇸")
	addRange(85, 64, 0, 0, 85, 127, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(85, 128, 0, 0, 85, 191, 255, 255, "RU", "Russia", "🇷🇺")
	addRange(85, 192, 0, 0, 85, 255, 255, 255, "NL", "Netherlands", "🇳🇱")

	// 86.0.0.0/8
	addCIDR(86, 0, 0, 0, 8, "GB", "United Kingdom", "🇬🇧")

	// 87.0.0.0/8
	addRange(87, 0, 0, 0, 87, 63, 255, 255, "IT", "Italy", "🇮🇹")
	addRange(87, 64, 0, 0, 87, 127, 255, 255, "ES", "Spain", "🇪🇸")
	addRange(87, 128, 0, 0, 87, 191, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(87, 192, 0, 0, 87, 255, 255, 255, "GB", "United Kingdom", "🇬🇧")

	// 88.0.0.0/8
	addRange(88, 0, 0, 0, 88, 63, 255, 255, "ES", "Spain", "🇪🇸")
	addRange(88, 64, 0, 0, 88, 127, 255, 255, "FR", "France", "🇫🇷")
	addRange(88, 128, 0, 0, 88, 191, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(88, 192, 0, 0, 88, 255, 255, 255, "IT", "Italy", "🇮🇹")

	// 89.0.0.0/8
	addRange(89, 0, 0, 0, 89, 63, 255, 255, "PL", "Poland", "🇵🇱")
	addRange(89, 64, 0, 0, 89, 127, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(89, 128, 0, 0, 89, 191, 255, 255, "RU", "Russia", "🇷🇺")
	addRange(89, 192, 0, 0, 89, 255, 255, 255, "IT", "Italy", "🇮🇹")

	// 90.0.0.0/8
	addRange(90, 0, 0, 0, 90, 127, 255, 255, "FR", "France", "🇫🇷")
	addRange(90, 128, 0, 0, 90, 255, 255, 255, "GB", "United Kingdom", "🇬🇧")

	// 91.0.0.0/8
	addRange(91, 0, 0, 0, 91, 63, 255, 255, "RU", "Russia", "🇷🇺")
	addRange(91, 64, 0, 0, 91, 127, 255, 255, "GB", "United Kingdom", "🇬🇧")
	addRange(91, 128, 0, 0, 91, 191, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(91, 192, 0, 0, 91, 255, 255, 255, "NL", "Netherlands", "🇳🇱")

	// 92.0.0.0/8
	addRange(92, 0, 0, 0, 92, 63, 255, 255, "GB", "United Kingdom", "🇬🇧")
	addRange(92, 64, 0, 0, 92, 127, 255, 255, "RU", "Russia", "🇷🇺")
	addRange(92, 128, 0, 0, 92, 191, 255, 255, "FR", "France", "🇫🇷")
	addRange(92, 192, 0, 0, 92, 255, 255, 255, "ES", "Spain", "🇪🇸")

	// 93.0.0.0/8
	addRange(93, 0, 0, 0, 93, 63, 255, 255, "IT", "Italy", "🇮🇹")
	addRange(93, 64, 0, 0, 93, 127, 255, 255, "ES", "Spain", "🇪🇸")
	addRange(93, 128, 0, 0, 93, 191, 255, 255, "FR", "France", "🇫🇷")
	addRange(93, 192, 0, 0, 93, 255, 255, 255, "RU", "Russia", "🇷🇺")

	// 94.0.0.0/8
	addRange(94, 0, 0, 0, 94, 63, 255, 255, "RU", "Russia", "🇷🇺")
	addRange(94, 64, 0, 0, 94, 127, 255, 255, "GB", "United Kingdom", "🇬🇧")
	addRange(94, 128, 0, 0, 94, 191, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(94, 192, 0, 0, 94, 255, 255, 255, "IT", "Italy", "🇮🇹")

	// 95.0.0.0/8
	addRange(95, 0, 0, 0, 95, 63, 255, 255, "RU", "Russia", "🇷🇺")
	addRange(95, 64, 0, 0, 95, 127, 255, 255, "ES", "Spain", "🇪🇸")
	addRange(95, 128, 0, 0, 95, 191, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(95, 192, 0, 0, 95, 255, 255, 255, "PL", "Poland", "🇵🇱")

	// 109.0.0.0/8
	addRange(109, 0, 0, 0, 109, 63, 255, 255, "RU", "Russia", "🇷🇺")
	addRange(109, 64, 0, 0, 109, 127, 255, 255, "GB", "United Kingdom", "🇬🇧")
	addRange(109, 128, 0, 0, 109, 191, 255, 255, "TR", "Turkey", "🇹🇷")
	addRange(109, 192, 0, 0, 109, 255, 255, 255, "DE", "Germany", "🇩🇪")

	// 141.0.0.0/8
	addCIDR(141, 0, 0, 0, 8, "DE", "Germany", "🇩🇪")

	// 176.0.0.0/8
	addRange(176, 0, 0, 0, 176, 63, 255, 255, "RU", "Russia", "🇷🇺")
	addRange(176, 64, 0, 0, 176, 127, 255, 255, "TR", "Turkey", "🇹🇷")
	addRange(176, 128, 0, 0, 176, 191, 255, 255, "UA", "Ukraine", "🇺🇦")
	addRange(176, 192, 0, 0, 176, 255, 255, 255, "DE", "Germany", "🇩🇪")

	// 178.0.0.0/8
	addRange(178, 0, 0, 0, 178, 63, 255, 255, "RU", "Russia", "🇷🇺")
	addRange(178, 64, 0, 0, 178, 127, 255, 255, "GB", "United Kingdom", "🇬🇧")
	addRange(178, 128, 0, 0, 178, 191, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(178, 192, 0, 0, 178, 255, 255, 255, "FR", "France", "🇫🇷")

	// 185.0.0.0/8
	addRange(185, 0, 0, 0, 185, 63, 255, 255, "NL", "Netherlands", "🇳🇱")
	addRange(185, 64, 0, 0, 185, 127, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(185, 128, 0, 0, 185, 191, 255, 255, "GB", "United Kingdom", "🇬🇧")
	addRange(185, 192, 0, 0, 185, 255, 255, 255, "FR", "France", "🇫🇷")

	// 188.0.0.0/8
	addRange(188, 0, 0, 0, 188, 63, 255, 255, "RU", "Russia", "🇷🇺")
	addRange(188, 64, 0, 0, 188, 127, 255, 255, "GB", "United Kingdom", "🇬🇧")
	addRange(188, 128, 0, 0, 188, 191, 255, 255, "IT", "Italy", "🇮🇹")
	addRange(188, 192, 0, 0, 188, 255, 255, 255, "DE", "Germany", "🇩🇪")

	// 193.0.0.0/8
	addRange(193, 0, 0, 0, 193, 127, 255, 255, "NL", "Netherlands", "🇳🇱")
	addRange(193, 128, 0, 0, 193, 255, 255, 255, "DE", "Germany", "🇩🇪")

	// 194.0.0.0/8
	addRange(194, 0, 0, 0, 194, 127, 255, 255, "GB", "United Kingdom", "🇬🇧")
	addRange(194, 128, 0, 0, 194, 255, 255, 255, "FR", "France", "🇫🇷")

	// 195.0.0.0/8
	addRange(195, 0, 0, 0, 195, 127, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(195, 128, 0, 0, 195, 255, 255, 255, "GB", "United Kingdom", "🇬🇧")

	// 212.0.0.0/8
	addRange(212, 0, 0, 0, 212, 127, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(212, 128, 0, 0, 212, 255, 255, 255, "GB", "United Kingdom", "🇬🇧")

	// 213.0.0.0/8
	addRange(213, 0, 0, 0, 213, 85, 255, 255, "FR", "France", "🇫🇷")
	addRange(213, 86, 0, 0, 213, 170, 255, 255, "DE", "Germany", "🇩🇪")
	addRange(213, 171, 0, 0, 213, 255, 255, 255, "ES", "Spain", "🇪🇸")

	// 217.0.0.0/8
	addCIDR(217, 0, 0, 0, 8, "DE", "Germany (Deutsche Telekom)", "🇩🇪")

	// Asia-Pacific (APNIC) - Mapped to concrete countries
	addCIDR(1, 0, 0, 0, 8, "AU", "Australia", "🇦🇺")
	addCIDR(14, 0, 0, 0, 8, "JP", "Japan", "🇯🇵")
	addCIDR(27, 0, 0, 0, 8, "IN", "India", "🇮🇳")
	addCIDR(36, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(39, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(42, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(43, 0, 0, 0, 8, "JP", "Japan", "🇯🇵")
	addCIDR(49, 0, 0, 0, 8, "IN", "India", "🇮🇳")
	addCIDR(58, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(59, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(60, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(61, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(101, 0, 0, 0, 8, "IN", "India", "🇮🇳")
	addCIDR(103, 0, 0, 0, 8, "SG", "Singapore", "🇸🇬")
	addCIDR(106, 0, 0, 0, 8, "JP", "Japan", "🇯🇵")
	addCIDR(110, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(111, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(112, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(113, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(114, 0, 0, 0, 8, "TW", "Taiwan", "🇹🇼")
	addCIDR(115, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(116, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(117, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(118, 0, 0, 0, 8, "JP", "Japan", "🇯🇵")
	addCIDR(119, 0, 0, 0, 8, "JP", "Japan", "🇯🇵")
	addCIDR(120, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(121, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(122, 0, 0, 0, 8, "JP", "Japan", "🇯🇵")
	addCIDR(123, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(124, 0, 0, 0, 8, "JP", "Japan", "🇯🇵")
	addCIDR(125, 0, 0, 0, 8, "JP", "Japan", "🇯🇵")
	addCIDR(126, 0, 0, 0, 8, "JP", "Japan (SoftBank)", "🇯🇵")
	addCIDR(153, 0, 0, 0, 8, "JP", "Japan", "🇯🇵")
	addCIDR(175, 0, 0, 0, 8, "KR", "South Korea", "🇰🇷")
	addCIDR(180, 0, 0, 0, 8, "JP", "Japan", "🇯🇵")
	addCIDR(182, 0, 0, 0, 8, "IN", "India", "🇮🇳")
	addCIDR(183, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(202, 0, 0, 0, 8, "AU", "Australia", "🇦🇺")
	addCIDR(203, 0, 0, 0, 8, "AU", "Australia", "🇦🇺")
	addCIDR(210, 0, 0, 0, 8, "JP", "Japan", "🇯🇵")
	addCIDR(211, 0, 0, 0, 8, "KR", "South Korea", "🇰🇷")
	addCIDR(218, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(219, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(220, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(221, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(222, 0, 0, 0, 8, "CN", "China", "🇨🇳")
	addCIDR(223, 0, 0, 0, 8, "IN", "India", "🇮🇳")

	// Latin America (LACNIC) - Mapped to concrete countries
	addCIDR(177, 0, 0, 0, 8, "BR", "Brazil", "🇧🇷")
	addCIDR(179, 0, 0, 0, 8, "BR", "Brazil", "🇧🇷")
	addCIDR(181, 0, 0, 0, 8, "BR", "Brazil", "🇧🇷")
	addCIDR(186, 0, 0, 0, 8, "AR", "Argentina", "🇦🇷")
	addCIDR(187, 0, 0, 0, 8, "BR", "Brazil", "🇧🇷")
	addCIDR(189, 0, 0, 0, 8, "MX", "Mexico", "🇲🇽")
	addCIDR(190, 0, 0, 0, 8, "CL", "Chile", "🇨🇱")
	addCIDR(191, 0, 0, 0, 8, "BR", "Brazil", "🇧🇷")
	addCIDR(200, 0, 0, 0, 8, "BR", "Brazil", "🇧🇷")
	addCIDR(201, 0, 0, 0, 8, "BR", "Brazil", "🇧🇷")

	// Africa (AFRINIC) - Mapped to concrete countries
	addCIDR(41, 0, 0, 0, 8, "ZA", "South Africa", "🇿🇦")
	addCIDR(102, 0, 0, 0, 8, "ZA", "South Africa", "🇿🇦")
	addCIDR(105, 0, 0, 0, 8, "EG", "Egypt", "🇪🇬")
	addCIDR(154, 0, 0, 0, 8, "NG", "Nigeria", "🇳🇬")
	addCIDR(196, 0, 0, 0, 8, "ZA", "South Africa", "🇿🇦")
	addCIDR(197, 0, 0, 0, 8, "KE", "Kenya", "🇰🇪")

	// Sort so narrower/more specific ranges precede broader ones
	sort.SliceStable(ranges, func(i, j int) bool {
		spanI := ranges[i].end - ranges[i].start
		spanJ := ranges[j].end - ranges[j].start
		if spanI != spanJ {
			return spanI < spanJ // More specific first!
		}
		return ranges[i].start < ranges[j].start
	})
}

// Lookup maps an IP address to its geographical and country origin.
func Lookup(addr netip.Addr) Location {
	loc := lookupRaw(addr)
	loc.Latitude, loc.Longitude = Coordinates(loc.CountryCode)
	return loc
}

func lookupRaw(addr netip.Addr) Location {
	if !addr.IsValid() || addr.IsUnspecified() {
		return Location{CountryCode: "LOCAL", CountryName: "Localhost", Flag: "🔒", IsLocal: true}
	}
	if addr.IsLoopback() {
		return Location{CountryCode: "HOST", CountryName: "On Host (Loopback)", Flag: "🏠", IsLocal: true}
	}
	if addr.IsPrivate() {
		return Location{CountryCode: "LAN", CountryName: "Local Network (LAN / Container)", Flag: "🔒", IsLocal: true}
	}
	if addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() {
		return Location{CountryCode: "LINK", CountryName: "Link-Local", Flag: "🔗", IsLocal: true}
	}

	// Unwrap IPv4-mapped IPv6 (e.g. ::ffff:1.2.3.4)
	if addr.Is4In6() {
		addr = addr.Unmap()
	}

	if addr.Is4() {
		b := addr.As4()
		val := binary.BigEndian.Uint32(b[:])
		for _, r := range ranges {
			if val >= r.start && val <= r.end {
				return Location{
					CountryCode: r.code,
					CountryName: r.name,
					Flag:        r.flag,
					IsLocal:     false,
				}
			}
		}
		return Location{CountryCode: "GLOBAL", CountryName: "International Internet", Flag: "🌐", IsLocal: false}
	}

	// IPv6 major ranges
	s := addr.String()
	switch {
	case strings.HasPrefix(s, "2001:4860:"):
		return Location{CountryCode: "US", CountryName: "United States (Google)", Flag: "🇺🇸", IsLocal: false}
	case strings.HasPrefix(s, "2606:4700:"):
		return Location{CountryCode: "US", CountryName: "United States (Cloudflare)", Flag: "🇺🇸", IsLocal: false}
	case strings.HasPrefix(s, "2a01:4f8:"), strings.HasPrefix(s, "2a01:4f9:"):
		return Location{CountryCode: "DE", CountryName: "Germany (Hetzner)", Flag: "🇩🇪", IsLocal: false}
	case strings.HasPrefix(s, "2001:67c:4e8:"):
		return Location{CountryCode: "NL", CountryName: "Netherlands (Telegram)", Flag: "🇳🇱", IsLocal: false}
	default:
		return Location{CountryCode: "GLOBAL", CountryName: "Global IPv6 Internet", Flag: "🌐", IsLocal: false}
	}
}

