package nearlyfreespeech

import (
	"testing"
	"time"

	"github.com/LukasDeco/lego/v4/platform/tester"
	"github.com/stretchr/testify/require"
)

const envDomain = envNamespace + "DOMAIN"

var envTest = tester.NewEnvTest(EnvAPIKey, EnvLogin).WithDomain(envDomain)

func TestNewDNSProvider(t *testing.T) {
	testCases := []struct {
		desc     string
		envVars  map[string]string
		expected string
	}{
		{
			desc: "success",
			envVars: map[string]string{
				EnvAPIKey: "123",
				EnvLogin:  "testuser",
			},
		},
		{
			desc: "missing credentials",
			envVars: map[string]string{
				EnvAPIKey: "",
				EnvLogin:  "",
			},
			expected: "nearlyfreespeech: some credentials information are missing: NEARLYFREESPEECH_API_KEY,NEARLYFREESPEECH_LOGIN",
		},
		{
			desc: "missing api key",
			envVars: map[string]string{
				EnvAPIKey: "",
				EnvLogin:  "testuser",
			},
			expected: "nearlyfreespeech: some credentials information are missing: NEARLYFREESPEECH_API_KEY",
		},
		{
			desc: "missing login",
			envVars: map[string]string{
				EnvAPIKey: "123",
				EnvLogin:  "",
			},
			expected: "nearlyfreespeech: some credentials information are missing: NEARLYFREESPEECH_LOGIN",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			defer envTest.RestoreEnv()
			envTest.ClearEnv()

			envTest.Apply(test.envVars)

			p, err := NewDNSProvider()

			if test.expected == "" {
				require.NoError(t, err)
				require.NotNil(t, p)
				require.NotNil(t, p.config)
			} else {
				require.EqualError(t, err, test.expected)
			}
		})
	}
}

func TestNewDNSProviderConfig(t *testing.T) {
	testCases := []struct {
		desc     string
		login    string
		apikey   string
		expected string
	}{
		{
			desc:   "success",
			login:  "login",
			apikey: "apikey",
		},
		{
			desc:     "missing credentials",
			expected: "nearlyfreespeech: API credentials are missing",
		},
		{
			desc:     "missing login",
			login:    "",
			apikey:   "apikey",
			expected: "nearlyfreespeech: API credentials are missing",
		},
		{
			desc:     "missing key",
			login:    "login",
			apikey:   "",
			expected: "nearlyfreespeech: API credentials are missing",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			config := NewDefaultConfig()
			config.APIKey = test.apikey
			config.Login = test.login

			p, err := NewDNSProviderConfig(config)

			if test.expected == "" {
				require.NoError(t, err)
				require.NotNil(t, p)
				require.NotNil(t, p.config)
			} else {
				require.EqualError(t, err, test.expected)
			}
		})
	}
}

func TestLivePresent(t *testing.T) {
	if !envTest.IsLiveTest() {
		t.Skip("skipping live test")
	}

	envTest.RestoreEnv()
	provider, err := NewDNSProvider()
	require.NoError(t, err)

	err = provider.Present(envTest.GetDomain(), "", "123d==")
	require.NoError(t, err)
}

func TestLiveCleanUp(t *testing.T) {
	if !envTest.IsLiveTest() {
		t.Skip("skipping live test")
	}

	envTest.RestoreEnv()
	provider, err := NewDNSProvider()
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	err = provider.CleanUp(envTest.GetDomain(), "", "123d==")
	require.NoError(t, err)
}
