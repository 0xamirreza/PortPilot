package geoip

// Coordinates returns the latitude and longitude for a given ISO country code or region.
func Coordinates(code string) (lat, lon float64) {
	switch code {
	case "IR":
		return 35.6892, 51.3890 // Iran (Tehran)
	case "US":
		return 38.8951, -77.0364 // United States
	case "DE":
		return 50.1109, 8.6821 // Germany (Frankfurt)
	case "NL":
		return 52.3676, 4.9041 // Netherlands (Amsterdam)
	case "FI":
		return 60.1699, 24.9384 // Finland (Helsinki)
	case "FR":
		return 48.8566, 2.3522 // France (Paris)
	case "GB":
		return 51.5074, -0.1278 // United Kingdom (London)
	case "CH":
		return 47.3769, 8.5417 // Switzerland (Zurich)
	case "SE":
		return 59.3293, 18.0686 // Sweden (Stockholm)
	case "SG":
		return 1.3521, 103.8198 // Singapore
	case "JP":
		return 35.6762, 139.6503 // Japan (Tokyo)
	case "CA":
		return 45.4215, -75.6972 // Canada (Ottawa)
	case "RU":
		return 55.7558, 37.6173 // Russia (Moscow)
	case "CN":
		return 39.9042, 116.4074 // China (Beijing)
	case "IN":
		return 28.6139, 77.2090 // India (New Delhi)
	case "BR":
		return -15.7975, -47.8919 // Brazil (Brasilia)
	case "AU":
		return -33.8688, 151.2093 // Australia (Sydney)
	case "IT":
		return 41.9028, 12.4964 // Italy (Rome)
	case "ES":
		return 40.4168, -3.7038 // Spain (Madrid)
	case "TR":
		return 39.9334, 32.8597 // Turkey (Ankara)
	case "AE":
		return 25.2048, 55.2708 // UAE (Dubai)
	case "KR":
		return 37.5665, 126.9780 // South Korea (Seoul)
	case "ZA":
		return -30.5595, 22.9375 // South Africa
	case "NO":
		return 59.9139, 10.7522 // Norway (Oslo)
	case "DK":
		return 55.6761, 12.5683 // Denmark (Copenhagen)
	case "PL":
		return 52.2297, 21.0122 // Poland (Warsaw)
	case "UA":
		return 50.4501, 30.5234 // Ukraine (Kyiv)
	case "AT":
		return 48.2082, 16.3738 // Austria (Vienna)
	case "BE":
		return 50.8503, 4.3517 // Belgium (Brussels)
	case "IE":
		return 53.3498, -6.2603 // Ireland (Dublin)
	case "IL":
		return 32.0853, 34.7818 // Israel (Tel Aviv)
	case "SA":
		return 24.7136, 46.6753 // Saudi Arabia (Riyadh)
	case "EG":
		return 30.0444, 31.2357 // Egypt (Cairo)
	case "LOCAL", "HOST", "LAN", "LINK":
		return 35.6892, 51.3890 // Local Host (Default Iran)
	default:
		return 25.0, 0.0 // Global Internet
	}
}

