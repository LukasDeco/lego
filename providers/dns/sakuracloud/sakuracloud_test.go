package sakuracloud

import (
	"testing"
	"time"

	"github.com/LukasDeco/lego/v4/platform/tester"
	"github.com/stretchr/testify/require"
)

const envDomain = envNamespace + "DOMAIN"

var envTest = tester.NewEnvTest(
	EnvAccessToken,
	EnvAccessTokenSecret).
	WithDomain(envDomain)

func TestNewDNSProvider(t *testing.T) {
	testCases := []struct {
		desc     string
		envVars  map[string]string
		expected string
	}{
		{
			desc: "success",
			envVars: map[string]string{
				EnvAccessToken:       "123",
				EnvAccessTokenSecret: "456",
			},
		},
		{
			desc: "missing credentials",
			envVars: map[string]string{
				EnvAccessToken:       "",
				EnvAccessTokenSecret: "",
			},
			expected: "sakuracloud: some credentials information are missing: SAKURACLOUD_ACCESS_TOKEN,SAKURACLOUD_ACCESS_TOKEN_SECRET",
		},
		{
			desc: "missing access token",
			envVars: map[string]string{
				EnvAccessToken:       "",
				EnvAccessTokenSecret: "456",
			},
			expected: "sakuracloud: some credentials information are missing: SAKURACLOUD_ACCESS_TOKEN",
		},
		{
			desc: "missing token secret",
			envVars: map[string]string{
				EnvAccessToken:       "123",
				EnvAccessTokenSecret: "",
			},
			expected: "sakuracloud: some credentials information are missing: SAKURACLOUD_ACCESS_TOKEN_SECRET",
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
				require.NotNil(t, p.client)
			} else {
				require.EqualError(t, err, test.expected)
			}
		})
	}
}

func TestNewDNSProviderConfig(t *testing.T) {
	testCases := []struct {
		desc     string
		token    string
		secret   string
		expected string
	}{
		{
			desc:   "success",
			token:  "123",
			secret: "456",
		},
		{
			desc:     "missing credentials",
			expected: "sakuracloud: AccessToken is missing",
		},
		{
			desc:     "missing token",
			secret:   "456",
			expected: "sakuracloud: AccessToken is missing",
		},
		{
			desc:     "missing secret",
			token:    "123",
			expected: "sakuracloud: AccessSecret is missing",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			config := NewDefaultConfig()
			config.Token = test.token
			config.Secret = test.secret

			p, err := NewDNSProviderConfig(config)

			if test.expected == "" {
				require.NoError(t, err)
				require.NotNil(t, p)
				require.NotNil(t, p.config)
				require.NotNil(t, p.client)
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
