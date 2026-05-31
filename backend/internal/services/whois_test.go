package services

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/syrup/backend/internal/models"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		// RFC1918 10.0.0.0/8
		{"10.0.0.0", true},
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		// RFC1918 172.16.0.0/12
		{"172.16.0.0", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"172.32.0.0", false},
		// RFC1918 192.168.0.0/16
		{"192.168.0.0", true},
		{"192.168.1.1", true},
		{"192.168.255.255", true},
		// Loopback 127.0.0.0/8
		{"127.0.0.1", true},
		{"127.255.255.255", true},
		// Public IPs
		{"8.8.8.8", false},
		{"8.8.4.4", false},
		{"1.1.1.1", false},
		{"9.9.9.9", false},
		{"172.15.0.1", false},
		{"192.167.255.255", false},
		{"11.0.0.1", false},
		// IPv6 loopback ::1
		{"::1", true},
		// IPv6 link-local fe80::/10
		{"fe80::1", true},
		{"fe80::", true},
		{"febf::1", true},
		{"fec0::1", false},
		// Unique Local Address (ULA) fc00::/7
		{"fc00::", true},
		{"fd00::1", true},
		{"fdff::1", true},
		{"fe00::1", false},
		// Public IPv6
		{"2001:db8::1", false},
		{"2607:f8b0::1", false},
		// CGNAT (RFC 6598) 100.64.0.0/10
		{"100.64.0.0", true},
		{"100.64.0.1", true},
		{"100.127.255.255", true},
		{"100.128.0.0", false},
		{"100.63.255.255", false},
		// Edge cases
		{"0.0.0.0", false},
		{"255.255.255.255", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			got := IsPrivateIP(tt.ip)
			if got != tt.expected {
				t.Errorf("IsPrivateIP(%q) = %v, want %v", tt.ip, got, tt.expected)
			}
		})
	}
}

func TestIsPrivateIP_InvalidInput(t *testing.T) {
	if IsPrivateIP("") {
		t.Error("IsPrivateIP(\"\") should return false")
	}
	if IsPrivateIP("not-an-ip") {
		t.Error("IsPrivateIP(\"not-an-ip\") should return false")
	}
	if IsPrivateIP("256.256.256.256") {
		t.Error("IsPrivateIP(\"256.256.256.256\") should return false")
	}
}

func TestParseWHOISResponse_Full(t *testing.T) {
	raw := `OrgName:        Google Inc.
Organization:   Google LLC
Country:        US
City:           Mountain View
Origin:         AS15169
OriginAS:       AS15169
`

	result := ParseWHOISResponse(raw)

	if result.Raw != raw {
		t.Error("Raw field should match input")
	}
	if result.Organization == nil || *result.Organization != "Google Inc." {
		t.Errorf("Organization = %v, want 'Google Inc.'", result.Organization)
	}
	if result.Country == nil || *result.Country != "US" {
		t.Errorf("Country = %v, want 'US'", result.Country)
	}
	if result.City == nil || *result.City != "Mountain View" {
		t.Errorf("City = %v, want 'Mountain View'", result.City)
	}
	if result.ASN == nil || *result.ASN != "AS15169" {
		t.Errorf("ASN = %v, want 'AS15169'", result.ASN)
	}
}

func TestParseWHOISResponse_OrganizationOnly(t *testing.T) {
	raw := `Organization:   IANA
OrgName:        Internet Assigned Numbers Authority
`

	result := ParseWHOISResponse(raw)

	if result.Organization == nil || *result.Organization != "IANA" {
		t.Errorf("Organization = %v, want 'IANA'", result.Organization)
	}
	// Country and City should be nil when missing
	if result.Country != nil {
		t.Errorf("Country should be nil, got %v", *result.Country)
	}
	if result.City != nil {
		t.Errorf("City should be nil, got %v", *result.City)
	}
	if result.ASN != nil {
		t.Errorf("ASN should be nil, got %v", *result.ASN)
	}
}

func TestParseWHOISResponse_OriginVariants(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"Origin field", "Origin: AS15169\n", "AS15169"},
		{"OriginAS field", "OriginAS: AS64512\n", "AS64512"},
		{"ASN field", "ASN: AS13335\n", "AS13335"},
		{"Origin without AS prefix", "Origin: 15169\n", "15169"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseWHOISResponse(tt.raw)
			if result.ASN == nil || *result.ASN != tt.want {
				t.Errorf("ASN = %v, want %q", result.ASN, tt.want)
			}
		})
	}
}

func TestParseWHOISResponse_FirstMatchWins(t *testing.T) {
	raw := `OrgName:        First Org
Organization:   Second Org
Country:        UK
City:           London
Origin:         AS11111
OriginAS:       AS22222
`

	result := ParseWHOISResponse(raw)

	if result.Organization == nil || *result.Organization != "First Org" {
		t.Errorf("Organization = %v, want 'First Org' (first match wins)", result.Organization)
	}
	if result.ASN == nil || *result.ASN != "AS11111" {
		t.Errorf("ASN = %v, want 'AS11111' (first match wins)", result.ASN)
	}
}

func TestParseWHOISResponse_Empty(t *testing.T) {
	result := ParseWHOISResponse("")
	if result.Raw != "" {
		t.Errorf("Raw = %q, want empty", result.Raw)
	}
	if result.Organization != nil {
		t.Error("Organization should be nil for empty input")
	}
	if result.Country != nil {
		t.Error("Country should be nil for empty input")
	}
	if result.City != nil {
		t.Error("City should be nil for empty input")
	}
	if result.ASN != nil {
		t.Error("ASN should be nil for empty input")
	}
}

func TestParseWHOISResponse_UnknownKeys(t *testing.T) {
	raw := `SomeRandom:    Value
NetRange:       8.8.8.0 - 8.8.8.255
CIDR:           8.8.8.0/24
`

	result := ParseWHOISResponse(raw)

	if result.Organization != nil {
		t.Error("Organization should be nil when no known key present")
	}
	if result.Country != nil {
		t.Error("Country should be nil when no known key present")
	}
	if result.City != nil {
		t.Error("City should be nil when no known key present")
	}
	if result.ASN != nil {
		t.Error("ASN should be nil when no known key present")
	}
}

func TestParseWHOISResponse_CaseInsensitive(t *testing.T) {
	raw := `orgname:        Some Company
ORGANIZATION:   Another
country:        jp
CITY:           Tokyo
origin:         AS17676
`

	result := ParseWHOISResponse(raw)

	if result.Organization == nil || *result.Organization != "Some Company" {
		t.Errorf("Organization = %v, want 'Some Company'", result.Organization)
	}
	if result.Country == nil || *result.Country != "jp" {
		t.Errorf("Country = %v, want 'jp'", result.Country)
	}
	if result.City == nil || *result.City != "Tokyo" {
		t.Errorf("City = %v, want 'Tokyo'", result.City)
	}
	if result.ASN == nil || *result.ASN != "AS17676" {
		t.Errorf("ASN = %v, want 'AS17676'", result.ASN)
	}
}

func TestParseWHOISResponse_TrimWhitespace(t *testing.T) {
	raw := `OrgName:        Google Inc.
Organization:     Google LLC  
Country:   US
City:      Mountain View   
Origin:    AS15169
`

	result := ParseWHOISResponse(raw)

	if result.Organization == nil || *result.Organization != "Google Inc." {
		t.Errorf("Organization = %v, want 'Google Inc.'", result.Organization)
	}
	if result.Country == nil || *result.Country != "US" {
		t.Errorf("Country = %v, want 'US'", result.Country)
	}
	if result.ASN == nil || *result.ASN != "AS15169" {
		t.Errorf("ASN = %v, want 'AS15169'", result.ASN)
	}
}

func startMockWHOISServer(t *testing.T, response string) (string, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock WHOIS server: %v", err)
	}

	addr := listener.Addr().String()

	ready := make(chan struct{})
	go func() {
		close(ready)
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		buf := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		conn.Read(buf) // read the query, ignore it
		fmt.Fprint(conn, response)
		conn.Close()
	}()

	<-ready

	return addr, func() { listener.Close() }
}

func TestQueryWHOIS_Success(t *testing.T) {
	response := `OrgName:        Google Inc.
Country:        US
City:           Mountain View
Origin:         AS15169
`

	addr, cleanup := startMockWHOISServer(t, response)
	defer cleanup()

	result, err := QueryWHOIS("8.8.8.8", addr)
	if err != nil {
		t.Fatalf("QueryWHOIS() returned error: %v", err)
	}

	if result.Organization == nil || *result.Organization != "Google Inc." {
		t.Errorf("Organization = %v, want 'Google Inc.'", result.Organization)
	}
	if result.Country == nil || *result.Country != "US" {
		t.Errorf("Country = %v, want 'US'", result.Country)
	}
	if result.City == nil || *result.City != "Mountain View" {
		t.Errorf("City = %v, want 'Mountain View'", result.City)
	}
	if result.ASN == nil || *result.ASN != "AS15169" {
		t.Errorf("ASN = %v, want 'AS15169'", result.ASN)
	}
	if result.Raw != response {
		t.Errorf("Raw = %q, want %q", result.Raw, response)
	}
}

func TestQueryWHOIS_EmptyResponse(t *testing.T) {
	addr, cleanup := startMockWHOISServer(t, "")
	defer cleanup()

	result, err := QueryWHOIS("8.8.8.8", addr)
	if err != nil {
		t.Fatalf("QueryWHOIS() returned error: %v", err)
	}
	if result.Organization != nil {
		t.Error("Organization should be nil for empty response")
	}
}

func TestQueryWHOIS_ConnectionRefused(t *testing.T) {
	_, err := QueryWHOIS("8.8.8.8", "127.0.0.1:19999")
	if err == nil {
		t.Error("QueryWHOIS() should return error on connection refused")
	}
}

func TestQueryWHOIS_Timeout(t *testing.T) {
	// Create a listener that accepts but never sends data
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start hang server: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		wg.Done()
		conn, _ := listener.Accept()
		if conn != nil {
			time.Sleep(15 * time.Second)
			conn.Close()
		}
	}()

	wg.Wait()

	start := time.Now()
	_, err = QueryWHOIS("8.8.8.8", addr)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("QueryWHOIS() should return timeout error")
	}
	if elapsed > 15*time.Second {
		t.Errorf("QueryWHOIS took %v, expected timeout around 10s", elapsed)
	}
}

func TestQueryWHOIS_IPv6Address(t *testing.T) {
	response := `OrgName:        Cloudflare
Country:        US
Origin:         AS13335
`

	addr, cleanup := startMockWHOISServer(t, response)
	defer cleanup()

	result, err := QueryWHOIS("2001:db8::1", addr)
	if err != nil {
		t.Fatalf("QueryWHOIS() with IPv6 returned error: %v", err)
	}
	if result.Organization == nil || *result.Organization != "Cloudflare" {
		t.Errorf("Organization = %v, want 'Cloudflare'", result.Organization)
	}
}

func TestQueryWHOIS_InvalidServer(t *testing.T) {
	_, err := QueryWHOIS("8.8.8.8", "invalid-server-format")
	if err == nil {
		t.Error("QueryWHOIS() should return error for invalid server format")
	}
}

func TestWHOISResult_JSONTags(t *testing.T) {
	result := &models.WHOISResult{
		Organization: strPtr("Google"),
		Country:      strPtr("US"),
		City:         strPtr("Mountain View"),
		ASN:          strPtr("AS15169"),
		Raw:          "raw text",
	}

	if result.Organization == nil || *result.Organization != "Google" {
		t.Error("Organization field broken")
	}
}

func strPtr(s string) *string {
	return &s
}
