package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	testMinimalConfig = `
		trust_domain = "TRUSTDOMAIN"
		cluster = "CLUSTER"
		server_socket_path = "SOCKETPATH"
`
	testMinimalInformerConfig = testMinimalConfig + `
        mode = "informer"
`
)

func TestLoadConfig(t *testing.T) {
	require := require.New(t)

	dir, err := ioutil.TempDir("", "spire-adm-webhook-config-")
	require.NoError(err)
	defer os.RemoveAll(dir)

	confPath := filepath.Join(dir, "test.conf")

	_, err = LoadConfig(confPath)
	require.Error(err)
	require.Contains(err.Error(), "unable to load configuration:")

	err = ioutil.WriteFile(confPath, []byte(testMinimalConfig), 0600)
	require.NoError(err)

	config, err := LoadConfig(confPath)
	require.NoError(err)

	require.Equal(&Config{
		LogLevel:         defaultLogLevel,
		Addr:             ":8443",
		CertPath:         defaultCertPath,
		KeyPath:          defaultKeyPath,
		CaCertPath:       defaultCaCertPath,
		ServerSocketPath: "SOCKETPATH",
		TrustDomain:      "TRUSTDOMAIN",
		Cluster:          "CLUSTER",
		Mode:             "admission",
	}, config)
}

func TestParseConfig(t *testing.T) {
	os.Setenv("KUBECONFIG", "DEFAULTKUBE")

	testCases := []struct {
		name string
		in   string
		out  *Config
		err  string
	}{
		{
			name: "admission defaults",
			in:   testMinimalConfig,
			out: &Config{
				LogLevel:                       defaultLogLevel,
				Addr:                           ":8443",
				CertPath:                       defaultCertPath,
				KeyPath:                        defaultKeyPath,
				CaCertPath:                     defaultCaCertPath,
				InsecureSkipClientVerification: false,
				ServerSocketPath:               "SOCKETPATH",
				TrustDomain:                    "TRUSTDOMAIN",
				Cluster:                        "CLUSTER",
				Mode:                           "admission",
			},
		},

		{
			name: "informer defaults",
			in:   testMinimalInformerConfig,
			out: &Config{
				LogLevel:               defaultLogLevel,
				ServerSocketPath:       "SOCKETPATH",
				TrustDomain:            "TRUSTDOMAIN",
				Cluster:                "CLUSTER",
				Mode:                   "informer",
				KubeConfig:             "DEFAULTKUBE",
				InformerResyncInterval: "0",
			},
		},

		{
			name: "overrides",
			in: `
				log_level = "LEVELOVERRIDE"
				log_path = "PATHOVERRIDE"
				addr = ":1234"
				cert_path = "CERTOVERRIDE"
				key_path = "KEYOVERRIDE"
				cacert_path = "CACERTOVERRIDE"
				insecure_skip_client_verification = true
				server_socket_path = "SOCKETPATHOVERRIDE"
				trust_domain = "TRUSTDOMAINOVERRIDE"
				cluster = "CLUSTEROVERRIDE"
				pod_label = "PODLABEL"
			`,
			out: &Config{
				LogLevel:                       "LEVELOVERRIDE",
				LogPath:                        "PATHOVERRIDE",
				Addr:                           ":1234",
				CertPath:                       "CERTOVERRIDE",
				KeyPath:                        "KEYOVERRIDE",
				CaCertPath:                     "CACERTOVERRIDE",
				InsecureSkipClientVerification: true,
				ServerSocketPath:               "SOCKETPATHOVERRIDE",
				TrustDomain:                    "TRUSTDOMAINOVERRIDE",
				Cluster:                        "CLUSTEROVERRIDE",
				PodLabel:                       "PODLABEL",
				Mode:                           "admission",
			},
		},

		{
			name: "informer overrides",
			in: `
				log_level = "LEVELOVERRIDE"
				log_path = "PATHOVERRIDE"
				server_socket_path = "SOCKETPATHOVERRIDE"
				trust_domain = "TRUSTDOMAINOVERRIDE"
				cluster = "CLUSTEROVERRIDE"
				pod_label = "PODLABEL"
                mode = "informer"
                kubeconfig = "KUBEOVERRIDE"
                informer_resync_interval = "10m"
            `,
			out: &Config{
				LogLevel:               "LEVELOVERRIDE",
				LogPath:                "PATHOVERRIDE",
				ServerSocketPath:       "SOCKETPATHOVERRIDE",
				TrustDomain:            "TRUSTDOMAINOVERRIDE",
				Cluster:                "CLUSTEROVERRIDE",
				PodLabel:               "PODLABEL",
				Mode:                   "informer",
				KubeConfig:             "KUBEOVERRIDE",
				InformerResyncInterval: "10m",
			},
		},

		{
			name: "bad HCL",
			in:   `INVALID`,
			err:  "unable to decode configuration",
		},
		{
			name: "missing server_socket_path",
			in: `
				trust_domain = "TRUSTDOMAIN"
				cluster = "CLUSTER"
			`,
			err: "server_socket_path must be specified",
		},
		{
			name: "missing trust domain",
			in: `
				cluster = "CLUSTER"
				server_socket_path = "SOCKETPATH"
			`,
			err: "trust_domain must be specified",
		},
		{
			name: "missing cluster",
			in: `
				trust_domain = "TRUSTDOMAIN"
				server_socket_path = "SOCKETPATH"
			`,
			err: "cluster must be specified",
		},
		{
			name: "workload registration mode specification is incorrect",
			in: testMinimalConfig + `
				pod_label = "PODLABEL"
				pod_annotation = "PODANNOTATION"
			`,
			err: "workload registration mode specification is incorrect, can't specify both pod_label and pod_annotation",
		},

		{
			name: "kubeconfig field in admission mode",
			in: testMinimalConfig + `
				kubeconfig = "KUBECONFIG"
			`,
			err: "kubeconfig not valid in admission mode",
		},
		{
			name: "addr field in informer mode",
			in: testMinimalInformerConfig + `
				addr = "ADDR"
			`,
			err: "addr not valid in informer mode",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase // alias loop variable as it is used in the closure
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := ParseConfig(testCase.in)
			if testCase.err != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), testCase.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testCase.out, actual)
		})
	}
}
