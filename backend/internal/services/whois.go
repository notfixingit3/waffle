package services

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/syrup/backend/internal/models"
)

func IsPrivateIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}

	ip4 := parsed.To4()
	if ip4 != nil {
		// RFC 1918: 10.0.0.0/8
		// Loopback: 127.0.0.0/8
		// RFC 1918: 172.16.0.0/12
		// RFC 1918: 192.168.0.0/16
		// CGNAT (RFC 6598): 100.64.0.0/10
		return ip4[0] == 10 ||
			ip4[0] == 127 ||
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
			(ip4[0] == 192 && ip4[1] == 168) ||
			(ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127)
	}

	// IPv6 loopback: ::1
	if parsed.Equal(net.IPv6loopback) {
		return true
	}

	// IPv6 link-local: fe80::/10
	if parsed[0] == 0xfe && parsed[1]&0xc0 == 0x80 {
		return true
	}

	// Unique Local Address (ULA): fc00::/7
	if parsed[0]&0xfe == 0xfc {
		return true
	}

	return false
}

func QueryWHOIS(ipAddress, server string) (*models.WHOISResult, error) {
	addr := server
	if _, _, err := net.SplitHostPort(server); err != nil {
		addr = net.JoinHostPort(server, "43")
	}

	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("whois dial %s: %w", addr, err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(10 * time.Second)) // #nosec G104

	fmt.Fprintf(conn, "%s\r\n", ipAddress)

	var sb strings.Builder
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if line != "" {
					sb.WriteString(line)
				}
				break
			}
			return nil, fmt.Errorf("whois read: %w", err)
		}
		sb.WriteString(line)
	}

	raw := sb.String()
	result := ParseWHOISResponse(raw)

	if result == nil {
		result = &models.WHOISResult{}
	}
	result.Raw = raw

	return result, nil
}

func ParseWHOISResponse(raw string) *models.WHOISResult {
	result := &models.WHOISResult{Raw: raw}

	if raw == "" {
		return result
	}

	orgFound := false
	asnFound := false

	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()

		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if value == "" {
			continue
		}

		keyLower := strings.ToLower(key)

		if !orgFound && (keyLower == "orgname" || keyLower == "organization") {
			result.Organization = &value
			orgFound = true
		}

		if result.Country == nil && keyLower == "country" {
			result.Country = &value
		}

		if result.City == nil && keyLower == "city" {
			result.City = &value
		}

		if !asnFound && (keyLower == "origin" || keyLower == "originas" || keyLower == "asn") {
			result.ASN = &value
			asnFound = true
		}
	}

	return result
}
