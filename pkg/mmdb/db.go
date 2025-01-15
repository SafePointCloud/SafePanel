package mmdb

import (
	"net/netip"

	"github.com/oschwald/maxminddb-golang/v2"
)

type MMDB struct {
	mmdb *maxminddb.Reader
}

type IPInfo struct {
	RegisteredCountry struct {
		IsoCode string `maxminddb:"iso_code"`
		Names   struct {
			En string `maxminddb:"en"`
		} `maxminddb:"names"`
	} `maxminddb:"registered_country"`
}

func NewMMDB(path string) (*MMDB, error) {
	mmdb, err := maxminddb.Open(path)
	if err != nil {
		return nil, err
	}
	return &MMDB{mmdb: mmdb}, nil
}

func (m *MMDB) Lookup(ip string) (*IPInfo, error) {
	var record IPInfo
	err := m.mmdb.Lookup(netip.MustParseAddr(ip)).Decode(&record)
	if err != nil {
		return nil, err
	}
	return &record, nil
}
