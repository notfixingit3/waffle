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
	if ip4 == nil {
		return false
	}

	return ip4[0] == 10 ||
		ip4[0] == 127 ||
		(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
		(ip4[0] == 192 && ip4[1] == 168)
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

	conn.SetDeadline(time.Now().Add(10 * time.Second))

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
