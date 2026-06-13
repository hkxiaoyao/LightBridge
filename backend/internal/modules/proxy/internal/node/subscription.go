package node

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/LightBridge/internal/util/urlvalidator"
)

const DefaultSubscriptionMaxBytes int64 = 2 << 20

type SubscriptionDownloadOptions struct {
	HTTPClient        *http.Client
	MaxBytes          int64
	AllowInsecureHTTP bool
	ResolveIP         func(context.Context, string) ([]net.IP, error)
}

func DownloadSubscription(ctx context.Context, rawURL string, opts SubscriptionDownloadOptions) ([]byte, error) {
	validated, err := validateSubscriptionURL(ctx, rawURL, opts.AllowInsecureHTTP, opts.ResolveIP)
	if err != nil {
		return nil, err
	}

	client := opts.HTTPClient
	if client == nil {
		client = newSubscriptionHTTPClient(opts.ResolveIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, validated, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/x-yaml,text/yaml,text/plain,*/*")
	req.Header.Set("User-Agent", "LightBridge-Proxy/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download subscription: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("subscription returned status %d", resp.StatusCode)
	}

	maxBytes := opts.MaxBytes
	if maxBytes <= 0 {
		maxBytes = DefaultSubscriptionMaxBytes
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read subscription: %w", err)
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("subscription response exceeds %d bytes", maxBytes)
	}
	if len(body) == 0 {
		return nil, errors.New("subscription response is empty")
	}

	return body, nil
}

func validateSubscriptionURL(ctx context.Context, rawURL string, allowInsecureHTTP bool, resolveIP func(context.Context, string) ([]net.IP, error)) (string, error) {
	validated, err := urlvalidator.ValidateHTTPURL(rawURL, allowInsecureHTTP, urlvalidator.ValidationOptions{
		AllowPrivate: false,
	})
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(validated)
	if err != nil {
		return "", fmt.Errorf("invalid subscription url: %w", err)
	}
	if err := validateSubscriptionPort(parsed); err != nil {
		return "", err
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if isBlockedSubscriptionHost(host) {
		return "", fmt.Errorf("subscription host is not allowed: %s", host)
	}
	if err := validateSubscriptionResolvedHost(ctx, host, resolveIP); err != nil {
		return "", err
	}
	return validated, nil
}

func newSubscriptionHTTPClient(resolveIP func(context.Context, string) ([]net.IP, error)) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ip, err := firstAllowedIP(ctx, host, resolveIP)
			if err != nil {
				return nil, err
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          20,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}

func validateSubscriptionResolvedHost(ctx context.Context, host string, resolveIP func(context.Context, string) ([]net.IP, error)) error {
	_, err := firstAllowedIP(ctx, host, resolveIP)
	return err
}

func firstAllowedIP(ctx context.Context, host string, resolveIP func(context.Context, string) ([]net.IP, error)) (net.IP, error) {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return nil, errors.New("subscription host is required")
	}
	if isBlockedSubscriptionHost(host) {
		return nil, fmt.Errorf("subscription host is not allowed: %s", host)
	}
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedSubscriptionIP(ip) {
			return nil, fmt.Errorf("subscription ip is not allowed: %s", ip.String())
		}
		return ip, nil
	}
	ips, err := lookupSubscriptionIPs(ctx, host, resolveIP)
	if err != nil {
		return nil, fmt.Errorf("resolve subscription host: %w", err)
	}
	if len(ips) == 0 {
		return nil, errors.New("subscription host resolved no addresses")
	}
	for _, ip := range ips {
		if isBlockedSubscriptionIP(ip) {
			return nil, fmt.Errorf("subscription resolved ip is not allowed: %s", ip.String())
		}
	}
	return ips[0], nil
}

func lookupSubscriptionIPs(ctx context.Context, host string, resolveIP func(context.Context, string) ([]net.IP, error)) ([]net.IP, error) {
	if resolveIP != nil {
		return resolveIP(ctx, host)
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	return ips, nil
}

func isBlockedSubscriptionHost(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	if host == "" {
		return true
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || host == "metadata.google.internal" {
		return true
	}
	if strings.HasSuffix(host, ".metadata.google.internal") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return isBlockedSubscriptionIP(ip)
	}
	return false
}

func isBlockedSubscriptionIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	return isIPv4MetadataIP(ip)
}

func isIPv4MetadataIP(ip net.IP) bool {
	v4 := ip.To4()
	return v4 != nil && v4[0] == 169 && v4[1] == 254 && v4[2] == 169 && v4[3] == 254
}

func subscriptionPort(parsed *url.URL) string {
	if port := parsed.Port(); port != "" {
		return port
	}
	if strings.EqualFold(parsed.Scheme, "https") {
		return "443"
	}
	return "80"
}

func validateSubscriptionPort(parsed *url.URL) error {
	port := subscriptionPort(parsed)
	num, err := strconv.Atoi(port)
	if err != nil || num <= 0 || num > 65535 {
		return fmt.Errorf("invalid subscription port: %s", port)
	}
	return nil
}
