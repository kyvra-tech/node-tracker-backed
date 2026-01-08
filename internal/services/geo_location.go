package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/models"
	"github.com/sirupsen/logrus"
)

// GeoLocationService handles IP geolocation lookups
type GeoLocationService struct {
	cache    map[string]*CachedLocation
	cacheMu  sync.RWMutex
	cacheTTL time.Duration
	client   *http.Client
	logger   *logrus.Logger
	apiURL   string
}

// CachedLocation stores cached geo data with timestamp
type CachedLocation struct {
	Location *models.GeoLocation
	CachedAt time.Time
}

// NewGeoLocationService creates a new geo location service
func NewGeoLocationService(logger *logrus.Logger) *GeoLocationService {
	return &GeoLocationService{
		cache:    make(map[string]*CachedLocation),
		cacheTTL: 7 * 24 * time.Hour, // 7 days cache
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
		apiURL: "http://ip-api.com/json",
	}
}

// GetLocation retrieves geo location for an IP address
func (s *GeoLocationService) GetLocation(ctx context.Context, ip string) (*models.GeoLocation, error) {
	// Check cache first
	s.cacheMu.RLock()
	if cached, ok := s.cache[ip]; ok {
		if time.Since(cached.CachedAt) < s.cacheTTL {
			s.cacheMu.RUnlock()
			return cached.Location, nil
		}
	}
	s.cacheMu.RUnlock()

	// Fetch from API
	url := fmt.Sprintf("%s/%s?fields=status,message,country,countryCode,region,regionName,city,zip,lat,lon,timezone,isp,org,as,query", s.apiURL, ip)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch geo data: %w", err)
	}
	defer resp.Body.Close()

	var geo models.GeoLocation
	if err := json.NewDecoder(resp.Body).Decode(&geo); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if geo.Status != "success" {
		return nil, fmt.Errorf("geo lookup failed: %s", geo.Status)
	}

	// Cache the result
	s.cacheMu.Lock()
	s.cache[ip] = &CachedLocation{
		Location: &geo,
		CachedAt: time.Now(),
	}
	s.cacheMu.Unlock()

	s.logger.WithFields(logrus.Fields{
		"ip":      ip,
		"country": geo.Country,
		"city":    geo.City,
	}).Debug("Resolved geo location")

	return &geo, nil
}

// BulkGetLocations retrieves geo locations for multiple IPs with rate limiting
func (s *GeoLocationService) BulkGetLocations(ctx context.Context, ips []string) (map[string]*models.GeoLocation, error) {
	results := make(map[string]*models.GeoLocation)

	// Rate limit: 45 requests per minute for free tier
	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()

	for _, ip := range ips {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		case <-ticker.C:
			geo, err := s.GetLocation(ctx, ip)
			if err != nil {
				s.logger.WithError(err).WithField("ip", ip).Warn("Failed to get geo location")
				continue
			}
			results[ip] = geo
		}
	}

	return results, nil
}

// ExtractIPFromAddress extracts IP address from various address formats
func (s *GeoLocationService) ExtractIPFromAddress(address string) string {
	// Handle multiaddr format: /ip4/192.168.1.1/tcp/21888/p2p/...
	if strings.HasPrefix(address, "/ip4/") {
		parts := strings.Split(address, "/")
		if len(parts) >= 3 {
			ip := parts[2]
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// Handle DNS multiaddr: /dns/example.com/tcp/21888/p2p/...
	if strings.HasPrefix(address, "/dns/") || strings.HasPrefix(address, "/dns4/") {
		parts := strings.Split(address, "/")
		if len(parts) >= 3 {
			host := parts[2]
			return s.resolveHost(host)
		}
	}

	// Handle URL format: https://rpc.example.com
	if strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://") {
		u, err := url.Parse(address)
		if err == nil {
			host := u.Hostname()
			if ip := net.ParseIP(host); ip != nil {
				return host
			}
			return s.resolveHost(host)
		}
	}

	// Handle host:port format: example.com:50051
	if strings.Contains(address, ":") {
		host, _, err := net.SplitHostPort(address)
		if err == nil {
			if ip := net.ParseIP(host); ip != nil {
				return host
			}
			return s.resolveHost(host)
		}
	}

	// Try to extract IP directly using regex
	ipv4Regex := regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
	if match := ipv4Regex.FindString(address); match != "" {
		if net.ParseIP(match) != nil {
			return match
		}
	}

	return ""
}

// resolveHost resolves a hostname to IP address
func (s *GeoLocationService) resolveHost(host string) string {
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		s.logger.WithError(err).WithField("host", host).Debug("Failed to resolve hostname")
		return ""
	}

	// Prefer IPv4
	for _, ip := range ips {
		if ip.To4() != nil {
			return ip.String()
		}
	}

	return ips[0].String()
}

// ClearCache clears the geo location cache
func (s *GeoLocationService) ClearCache() {
	s.cacheMu.Lock()
	s.cache = make(map[string]*CachedLocation)
	s.cacheMu.Unlock()
}

// GetCacheStats returns cache statistics
func (s *GeoLocationService) GetCacheStats() (total int, valid int) {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	total = len(s.cache)
	for _, cached := range s.cache {
		if time.Since(cached.CachedAt) < s.cacheTTL {
			valid++
		}
	}
	return
}

// LookupAddress extracts IP from an address and performs geo lookup
func (s *GeoLocationService) LookupAddress(ctx context.Context, address string) (*models.GeoLocation, error) {
	ip := s.ExtractIPFromAddress(address)
	if ip == "" {
		return nil, fmt.Errorf("could not extract IP from address: %s", address)
	}
	return s.GetLocation(ctx, ip)
}
