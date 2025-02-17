// Package mythicbeasts implements a DNS provider for solving the DNS-01 challenge using Mythic Beasts API.
package mythicbeasts

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/LukasDeco/lego/v4/challenge/dns01"
	"github.com/LukasDeco/lego/v4/platform/config/env"
)

// Environment variables names.
const (
	envNamespace = "MYTHICBEASTS_"

	EnvUserName        = envNamespace + "USERNAME"
	EnvPassword        = envNamespace + "PASSWORD"
	EnvAPIEndpoint     = envNamespace + "API_ENDPOINT"
	EnvAuthAPIEndpoint = envNamespace + "AUTH_API_ENDPOINT"

	EnvTTL                = envNamespace + "TTL"
	EnvPropagationTimeout = envNamespace + "PROPAGATION_TIMEOUT"
	EnvPollingInterval    = envNamespace + "POLLING_INTERVAL"
	EnvHTTPTimeout        = envNamespace + "HTTP_TIMEOUT"
)

// Config is used to configure the creation of the DNSProvider.
type Config struct {
	UserName           string
	Password           string
	HTTPClient         *http.Client
	PropagationTimeout time.Duration
	PollingInterval    time.Duration
	APIEndpoint        *url.URL
	AuthAPIEndpoint    *url.URL
	TTL                int
}

// NewDefaultConfig returns a default configuration for the DNSProvider.
func NewDefaultConfig() (*Config, error) {
	apiEndpoint, err := url.Parse(env.GetOrDefaultString(EnvAPIEndpoint, apiBaseURL))
	if err != nil {
		return nil, fmt.Errorf("mythicbeasts: Unable to parse API URL: %w", err)
	}

	authEndpoint, err := url.Parse(env.GetOrDefaultString(EnvAuthAPIEndpoint, authBaseURL))
	if err != nil {
		return nil, fmt.Errorf("mythicbeasts: Unable to parse AUTH API URL: %w", err)
	}

	return &Config{
		TTL:                env.GetOrDefaultInt(EnvTTL, dns01.DefaultTTL),
		PropagationTimeout: env.GetOrDefaultSecond(EnvPropagationTimeout, dns01.DefaultPropagationTimeout),
		PollingInterval:    env.GetOrDefaultSecond(EnvPollingInterval, dns01.DefaultPollingInterval),
		APIEndpoint:        apiEndpoint,
		AuthAPIEndpoint:    authEndpoint,
		HTTPClient: &http.Client{
			Timeout: env.GetOrDefaultSecond(EnvHTTPTimeout, 10*time.Second),
		},
	}, nil
}

// DNSProvider implements the challenge.Provider interface.
type DNSProvider struct {
	config *Config

	// token  string
	token   *authResponse
	muToken sync.Mutex
}

// NewDNSProvider returns a DNSProvider instance configured for mythicbeasts DNSv2 API.
// Credentials must be passed in the environment variables:
// MYTHICBEASTS_USERNAME and MYTHICBEASTS_PASSWORD.
func NewDNSProvider() (*DNSProvider, error) {
	values, err := env.Get(EnvUserName, EnvPassword)
	if err != nil {
		return nil, fmt.Errorf("mythicbeasts: %w", err)
	}

	config, err := NewDefaultConfig()
	if err != nil {
		return nil, fmt.Errorf("mythicbeasts: %w", err)
	}
	config.UserName = values[EnvUserName]
	config.Password = values[EnvPassword]

	return NewDNSProviderConfig(config)
}

// NewDNSProviderConfig return a DNSProvider instance configured for mythicbeasts DNSv2 API.
func NewDNSProviderConfig(config *Config) (*DNSProvider, error) {
	if config == nil {
		return nil, errors.New("mythicbeasts: the configuration of the DNS provider is nil")
	}

	if config.UserName == "" || config.Password == "" {
		return nil, errors.New("mythicbeasts: incomplete credentials, missing username and/or password")
	}

	return &DNSProvider{config: config}, nil
}

// Present creates a TXT record using the specified parameters.
func (d *DNSProvider) Present(domain, token, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)

	authZone, err := dns01.FindZoneByFqdn(info.EffectiveFQDN)
	if err != nil {
		return fmt.Errorf("mythicbeasts: %w", err)
	}

	subDomain, err := dns01.ExtractSubDomain(info.EffectiveFQDN, authZone)
	if err != nil {
		return fmt.Errorf("mythicbeasts: %w", err)
	}

	authZone = dns01.UnFqdn(authZone)

	err = d.login()
	if err != nil {
		return fmt.Errorf("mythicbeasts: %w", err)
	}

	err = d.createTXTRecord(authZone, subDomain, info.Value)
	if err != nil {
		return fmt.Errorf("mythicbeasts: %w", err)
	}

	return nil
}

// CleanUp removes the TXT record matching the specified parameters.
func (d *DNSProvider) CleanUp(domain, token, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)

	authZone, err := dns01.FindZoneByFqdn(info.EffectiveFQDN)
	if err != nil {
		return fmt.Errorf("mythicbeasts: %w", err)
	}

	subDomain, err := dns01.ExtractSubDomain(info.EffectiveFQDN, authZone)
	if err != nil {
		return fmt.Errorf("mythicbeasts: %w", err)
	}

	authZone = dns01.UnFqdn(authZone)

	err = d.login()
	if err != nil {
		return fmt.Errorf("mythicbeasts: %w", err)
	}

	err = d.removeTXTRecord(authZone, subDomain, info.Value)
	if err != nil {
		return fmt.Errorf("mythicbeasts: %w", err)
	}

	return nil
}

// Timeout returns the timeout and interval to use when checking for DNS propagation.
// Adjusting here to cope with spikes in propagation times.
func (d *DNSProvider) Timeout() (timeout, interval time.Duration) {
	return d.config.PropagationTimeout, d.config.PollingInterval
}
