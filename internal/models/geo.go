package models

// GeoLocation represents geographic location data
type GeoLocation struct {
	Status      string  `json:"status"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Zip         string  `json:"zip"`
	Latitude    float64 `json:"lat"`
	Longitude   float64 `json:"lon"`
	Timezone    string  `json:"timezone"`
	ISP         string  `json:"isp"`
	Org         string  `json:"org"`
	AS          string  `json:"as"`
	Query       string  `json:"query"`
}

// IsValid checks if the geo location data is valid
func (g *GeoLocation) IsValid() bool {
	return g.Status == "success" && g.Country != ""
}
