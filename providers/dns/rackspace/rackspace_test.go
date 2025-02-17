package rackspace

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/LukasDeco/lego/v4/platform/tester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const envDomain = envNamespace + "DOMAIN"

var envTest = tester.NewEnvTest(
	EnvUser,
	EnvAPIKey).
	WithDomain(envDomain)

func TestNewDNSProviderConfig(t *testing.T) {
	config := setupTest(t)

	provider, err := NewDNSProviderConfig(config)
	require.NoError(t, err)
	assert.NotNil(t, provider.config)

	assert.Equal(t, provider.token, "testToken", "The token should match")
}

func TestNewDNSProviderConfig_MissingCredErr(t *testing.T) {
	_, err := NewDNSProviderConfig(NewDefaultConfig())
	assert.EqualError(t, err, "rackspace: credentials missing")
}

func TestDNSProvider_Present(t *testing.T) {
	config := setupTest(t)

	provider, err := NewDNSProviderConfig(config)

	if assert.NoError(t, err) {
		err = provider.Present("example.com", "token", "keyAuth")
		require.NoError(t, err)
	}
}

func TestDNSProvider_CleanUp(t *testing.T) {
	config := setupTest(t)

	provider, err := NewDNSProviderConfig(config)

	if assert.NoError(t, err) {
		err = provider.CleanUp("example.com", "token", "keyAuth")
		require.NoError(t, err)
	}
}

func TestLiveNewDNSProvider_ValidEnv(t *testing.T) {
	if !envTest.IsLiveTest() {
		t.Skip("skipping live test")
	}

	envTest.RestoreEnv()
	provider, err := NewDNSProvider()
	require.NoError(t, err)

	assert.Contains(t, provider.cloudDNSEndpoint, "https://dns.api.rackspacecloud.com/v1.0/", "The endpoint URL should contain the base")
}

func TestLivePresent(t *testing.T) {
	if !envTest.IsLiveTest() {
		t.Skip("skipping live test")
	}

	envTest.RestoreEnv()
	provider, err := NewDNSProvider()
	require.NoError(t, err)

	err = provider.Present(envTest.GetDomain(), "", "112233445566==")
	require.NoError(t, err)
}

func TestLiveCleanUp(t *testing.T) {
	if !envTest.IsLiveTest() {
		t.Skip("skipping live test")
	}

	envTest.RestoreEnv()
	provider, err := NewDNSProvider()
	require.NoError(t, err)

	time.Sleep(15 * time.Second)

	err = provider.CleanUp(envTest.GetDomain(), "", "112233445566==")
	require.NoError(t, err)
}

func setupTest(t *testing.T) *Config {
	t.Helper()

	dnsAPI := httptest.NewServer(dnsHandler())
	t.Cleanup(dnsAPI.Close)

	identityAPI := httptest.NewServer(identityHandler(dnsAPI.URL + "/123456"))
	t.Cleanup(identityAPI.Close)

	config := NewDefaultConfig()
	config.APIUser = "testUser"
	config.APIKey = "testKey"
	config.HTTPClient = identityAPI.Client()
	config.BaseURL = identityAPI.URL + "/"

	return config
}

func identityHandler(dnsEndpoint string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if string(reqBody) != `{"auth":{"RAX-KSKEY:apiKeyCredentials":{"username":"testUser","apiKey":"testKey"}}}` {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		resp := strings.Replace(identityResponseMock, "https://dns.api.rackspacecloud.com/v1.0/123456", dnsEndpoint, 1)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, resp)
	})
}

func dnsHandler() *http.ServeMux {
	mux := http.NewServeMux()

	// Used by `getHostedZoneID()` finding `zoneID` "?name=example.com"
	mux.HandleFunc("/123456/domains", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("name") == "example.com" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, zoneDetailsMock)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	})

	mux.HandleFunc("/123456/domains/112233/records", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		// Used by `Present()` creating the TXT record
		case http.MethodPost:
			reqBody, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if string(reqBody) != `{"records":[{"name":"_acme-challenge.example.com","type":"TXT","data":"pW9ZKG0xz_PCriK-nCMOjADy9eJcgGWIzkkj2fN4uZM","ttl":300}]}` {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, recordResponseMock)
			// Used by `findTxtRecord()` finding `record.ID` "?type=TXT&name=_acme-challenge.example.com"
		case http.MethodGet:
			if r.URL.Query().Get("type") == "TXT" && r.URL.Query().Get("name") == "_acme-challenge.example.com" {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, recordDetailsMock)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			return
			// Used by `CleanUp()` deleting the TXT record "?id=445566"
		case http.MethodDelete:
			if r.URL.Query().Get("id") == "TXT-654321" {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, recordDeleteMock)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
		}
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Printf("Not Found for Request: (%+v)\n\n", r)
	})

	return mux
}
