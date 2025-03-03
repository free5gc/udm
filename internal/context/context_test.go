package context

import (
	"net/netip"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/free5gc/openapi/models"
	"github.com/free5gc/udm/pkg/factory"
)

func createConfigFile(t *testing.T, postContent []byte) *os.File {
	content := []byte(`info:
  version: "1.0.4"

logger:
  level: info
`)

	configFile, err := os.CreateTemp("", "")
	if err != nil {
		t.Errorf("can't create temp file: %+v", err)
	}

	if _, err := configFile.Write(content); err != nil {
		t.Errorf("can't write content of temp file: %+v", err)
	}
	if _, err := configFile.Write(postContent); err != nil {
		t.Errorf("can't write content of temp file: %+v", err)
	}
	if err := configFile.Close(); err != nil {
		t.Fatal(err)
	}
	return configFile
}

func TestInitUdmContextWithConfigIPv6(t *testing.T) {
	postContent := []byte(`configuration:
  serviceNameList:
    - nudm-sdm
  sbi:
    scheme: http
    registerIP: "2001:db8::1:0:0:13"
    bindingIP: "2001:db8::1:0:0:13"
    port: 8313
  nrfUri: http://127.0.0.10:8000`)

	configFile := createConfigFile(t, postContent)

	// Test the initialization with the config file
	cfg, err := factory.ReadConfig(configFile.Name())
	if err != nil {
		t.Errorf("invalid read config: %+v %+v", err, cfg)
	}
	factory.UdmConfig = cfg

	GetSelf().NfService = make(map[models.ServiceName]models.NrfNfManagementNfService)
	InitUdmContext(GetSelf())

	assert.Equal(t, udmContext.SBIPort, 8313)
	assert.Equal(t, udmContext.RegisterIP.String(), "2001:db8::1:0:0:13")
	assert.Equal(t, udmContext.BindingIP.String(), "2001:db8::1:0:0:13")
	assert.Equal(t, udmContext.UriScheme, models.UriScheme("http"))

	// Close the config file
	t.Cleanup(func() {
		if err := os.RemoveAll(configFile.Name()); err != nil {
			t.Fatal(err)
		}
	})
}

func TestInitUdmContextWithConfigIPv4(t *testing.T) {
	postContent := []byte(`configuration:
  serviceNameList:
    - nudm-sdm
  sbi:
    scheme: http
    registerIP: "127.0.0.13"
    bindingIP: "127.0.0.13"
    port: 8131
  nrfUri: http://127.0.0.10:8000`)

	configFile := createConfigFile(t, postContent)

	// Test the initialization with the config file
	cfg, err := factory.ReadConfig(configFile.Name())
	if err != nil {
		t.Errorf("invalid read config: %+v %+v", err, cfg)
	}
	factory.UdmConfig = cfg

	GetSelf().NfService = make(map[models.ServiceName]models.NrfNfManagementNfService)
	InitUdmContext(GetSelf())

	assert.Equal(t, udmContext.SBIPort, 8131)
	assert.Equal(t, udmContext.RegisterIP.String(), "127.0.0.13")
	assert.Equal(t, udmContext.BindingIP.String(), "127.0.0.13")
	assert.Equal(t, udmContext.UriScheme, models.UriScheme("http"))

	// Close the config file
	t.Cleanup(func() {
		if err := os.RemoveAll(configFile.Name()); err != nil {
			t.Fatal(err)
		}
	})
}

func TestInitUdmContextWithConfigDeprecated(t *testing.T) {
	postContent := []byte(`configuration:
  serviceNameList:
    - nudm-sdm
  sbi:
    scheme: http
    registerIPv4: "127.0.0.30"
    bindingIPv4: "127.0.0.30"
    port: 8003
  nrfUri: http://127.0.0.10:8000`)

	configFile := createConfigFile(t, postContent)

	// Test the initialization with the config file
	cfg, err := factory.ReadConfig(configFile.Name())
	if err != nil {
		t.Errorf("invalid read config: %+v %+v", err, cfg)
	}
	factory.UdmConfig = cfg

	GetSelf().NfService = make(map[models.ServiceName]models.NrfNfManagementNfService)
	InitUdmContext(GetSelf())

	assert.Equal(t, udmContext.SBIPort, 8003)
	assert.Equal(t, udmContext.RegisterIP.String(), "127.0.0.30")
	assert.Equal(t, udmContext.BindingIP.String(), "127.0.0.30")
	assert.Equal(t, udmContext.UriScheme, models.UriScheme("http"))

	// Close the config file
	t.Cleanup(func() {
		if err := os.RemoveAll(configFile.Name()); err != nil {
			t.Fatal(err)
		}
	})
}

func TestInitUdmContextWithConfigEmptySBI(t *testing.T) {
	postContent := []byte(`configuration:
  serviceNameList:
    - nudm-sdm
  nrfUri: http://127.0.0.10:8000`)

	configFile := createConfigFile(t, postContent)

	// Test the initialization with the config file
	cfg, err := factory.ReadConfig(configFile.Name())
	if err != nil {
		t.Errorf("invalid read config: %+v %+v", err, cfg)
	}
	factory.UdmConfig = cfg

	GetSelf().NfService = make(map[models.ServiceName]models.NrfNfManagementNfService)
	InitUdmContext(GetSelf())

	assert.Equal(t, udmContext.SBIPort, 8000)
	assert.Equal(t, udmContext.RegisterIP.String(), "127.0.0.3")
	assert.Equal(t, udmContext.BindingIP.String(), "127.0.0.3")
	assert.Equal(t, udmContext.UriScheme, models.UriScheme("https"))

	// Close the config file
	t.Cleanup(func() {
		if err := os.RemoveAll(configFile.Name()); err != nil {
			t.Fatal(err)
		}
	})
}

func TestInitUdmContextWithConfigMissingRegisterIP(t *testing.T) {
	postContent := []byte(`configuration:
  sbi:
    bindingIP: "2001:db8::1:0:0:130"
  serviceNameList:
    - nudm-sdm
  nrfUri: http://127.0.0.10:8000`)

	configFile := createConfigFile(t, postContent)

	// Test the initialization with the config file
	cfg, err := factory.ReadConfig(configFile.Name())
	if err != nil {
		t.Errorf("invalid read config: %+v %+v", err, cfg)
	}
	factory.UdmConfig = cfg

	GetSelf().NfService = make(map[models.ServiceName]models.NrfNfManagementNfService)
	InitUdmContext(GetSelf())

	assert.Equal(t, udmContext.SBIPort, 8000)
	assert.Equal(t, udmContext.BindingIP.String(), "2001:db8::1:0:0:130")
	assert.Equal(t, udmContext.RegisterIP.String(), "2001:db8::1:0:0:130")
	assert.Equal(t, udmContext.UriScheme, models.UriScheme("https"))

	// Close the config file
	t.Cleanup(func() {
		if err := os.RemoveAll(configFile.Name()); err != nil {
			t.Fatal(err)
		}
	})
}

func TestInitUdmContextWithConfigMissingBindingIP(t *testing.T) {
	postContent := []byte(`configuration:
  sbi:
    registerIP: "2001:db8::1:0:0:131"
  serviceNameList:
    - nudm-sdm
  nrfUri: http://127.0.0.10:8000`)

	configFile := createConfigFile(t, postContent)

	// Test the initialization with the config file
	cfg, err := factory.ReadConfig(configFile.Name())
	if err != nil {
		t.Errorf("invalid read config: %+v %+v", err, cfg)
	}
	factory.UdmConfig = cfg

	GetSelf().NfService = make(map[models.ServiceName]models.NrfNfManagementNfService)
	InitUdmContext(GetSelf())

	assert.Equal(t, udmContext.SBIPort, 8000)
	assert.Equal(t, udmContext.BindingIP.String(), "2001:db8::1:0:0:131")
	assert.Equal(t, udmContext.RegisterIP.String(), "2001:db8::1:0:0:131")
	assert.Equal(t, udmContext.UriScheme, models.UriScheme("https"))

	// Close the config file
	t.Cleanup(func() {
		if err := os.RemoveAll(configFile.Name()); err != nil {
			t.Fatal(err)
		}
	})
}

func TestInitUdmContextWithConfigIPv6FromEnv(t *testing.T) {
	postContent := []byte(`configuration:
  serviceNameList:
    - nudm-sdm
  sbi:
    scheme: http
    registerIP: "MY_REGISTER_IP"
    bindingIP: "MY_BINDING_IP"
    port: 8313
  nrfUri: http://127.0.0.10:8000`)

	configFile := createConfigFile(t, postContent)

	if err := os.Setenv("MY_REGISTER_IP", "2001:db8::1:0:0:130"); err != nil {
		t.Errorf("Can't set MY_BINDING_IP variable environnement: %+v", err)
	}
	if err := os.Setenv("MY_BINDING_IP", "2001:db8::1:0:0:130"); err != nil {
		t.Errorf("Can't set MY_BINDING_IP variable environnement: %+v", err)
	}

	// Test the initialization with the config file
	cfg, err := factory.ReadConfig(configFile.Name())
	if err != nil {
		t.Errorf("invalid read config: %+v %+v", err, cfg)
	}
	factory.UdmConfig = cfg

	GetSelf().NfService = make(map[models.ServiceName]models.NrfNfManagementNfService)
	InitUdmContext(GetSelf())

	assert.Equal(t, udmContext.SBIPort, 8313)
	assert.Equal(t, udmContext.RegisterIP.String(), "2001:db8::1:0:0:130")
	assert.Equal(t, udmContext.BindingIP.String(), "2001:db8::1:0:0:130")
	assert.Equal(t, udmContext.UriScheme, models.UriScheme("http"))

	// Close the config file
	t.Cleanup(func() {
		if err := os.RemoveAll(configFile.Name()); err != nil {
			t.Fatal(err)
		}
	})
}

func TestResolveIPLocalhost(t *testing.T) {
	expectedAddr, err := netip.ParseAddr("::1")
	if err != nil {
		t.Errorf("invalid expected IP: %+v", expectedAddr)
	}

	addr := resolveIP("localhost")
	if addr != expectedAddr {
		t.Errorf("invalid IP: %+v", addr)
	}
	assert.Equal(t, addr, expectedAddr)
}

func TestResolveIPv4(t *testing.T) {
	expectedAddr, err := netip.ParseAddr("127.0.0.1")
	if err != nil {
		t.Errorf("invalid expected IP: %+v", expectedAddr)
	}

	addr := resolveIP("127.0.0.1")
	if addr != expectedAddr {
		t.Errorf("invalid IP: %+v", addr)
	}
}

func TestResolveIPv6(t *testing.T) {
	expectedAddr, err := netip.ParseAddr("2001:db8::1:0:0:1")
	if err != nil {
		t.Errorf("invalid expected IP: %+v", expectedAddr)
	}

	addr := resolveIP("2001:db8::1:0:0:1")
	if addr != expectedAddr {
		t.Errorf("invalid IP: %+v", addr)
	}
}
