package provider

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// NowDNSProvider implements the Provider interface for now-dns.com.
type NowDNSProvider struct {
	email    string
	password string
	client   *http.Client
	endpoint string
}

// NewNowDNSProvider creates a new NowDNSProvider instance.
func NewNowDNSProvider(email, password string) *NowDNSProvider {
	return &NowDNSProvider{
		email:    email,
		password: password,
		client:   http.DefaultClient,
		endpoint: "https://now-dns.com/update",
	}
}

// UpdateRecord updates the DNS record for the given domain to the specified IP.
func (p *NowDNSProvider) UpdateRecord(domain string, ip string) error {
	// Build request URL with properly escaped parameters
	reqURL := fmt.Sprintf("%s?hostname=%s&myip=%s", p.endpoint, url.QueryEscape(domain), url.QueryEscape(ip))

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set Basic Auth credentials
	req.SetBasicAuth(p.email, p.password)

	// Execute the request
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Parse and handle response
	response := strings.TrimSpace(string(body))

	// Handle response codes from now-dns.com API
	switch {
	case strings.HasPrefix(response, "good"):
		// Update succeeded
		return nil
	case strings.HasPrefix(response, "nochg"):
		// IP did not change - this is still a success
		return nil
	case strings.HasPrefix(response, "nohost"):
		return errors.New("host supplied not valid for given user")
	case strings.HasPrefix(response, "notfqdn"):
		return errors.New("host supplied is not a valid hostname")
	case strings.HasPrefix(response, "badauth"):
		return errors.New("invalid credentials")
	default:
		return fmt.Errorf("unexpected response from now-dns.com: %s", response)
	}
}
