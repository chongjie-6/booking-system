package geoip

import (
	"log"
	"net/netip"
	"os"

	"github.com/oschwald/maxminddb-golang/v2"
)

var reader *maxminddb.Reader

// Open loads the IP-to-country database from GEOIP_DB_PATH, or data/dbip-country-lite.mmdb if unset.
// Takes nothing; returns nothing, and only logs if the file is missing, since Country then returns "" for every IP.
func Open() {
	path := os.Getenv("GEOIP_DB_PATH")
	if path == "" {
		path = "data/dbip-country-lite.mmdb"
	}
	r, err := maxminddb.Open(path)
	if err != nil {
		log.Printf("geoip database unavailable, login country check disabled: %v", err)
		return
	}
	reader = r
}

// Country returns the ISO country code for addr, such as "AU". Takes an IP; returns "" if the IP isn't in the database
// (private and loopback addresses never are) or the database isn't loaded.
func Country(addr netip.Addr) string {
	if reader == nil {
		return ""
	}
	var code string
	if err := reader.Lookup(addr).DecodePath(&code, "country", "iso_code"); err != nil {
		return ""
	}
	return code
}
