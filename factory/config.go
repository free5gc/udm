/*
 * UDM Configuration Factory
 */

package factory

type Config struct {
	Info *Info `yaml:"info"`

	Configuration *Configuration `yaml:"configuration"`
}

type Info struct {
	Version string `yaml:"version,omitempty"`

	Description string `yaml:"description,omitempty"`
}

type Configuration struct {
	UdmName string `yaml:"udmName,omitempty"`

	Sbi *Sbi `yaml:"sbi,omitempty"`

	ServiceNameList []string `yaml:"serviceNameList,omitempty"`

	Udrclient *Udrclient `yaml:"udrclient,omitempty"`

	Nrfclient *Nrfclient `yaml:"nrfclient,omitempty"`

	NrfUri string `yaml:"nrfUri,omitempty"`

	Keys *Keys `yaml:"keys,omitempty"`
}

type Sbi struct {
	Scheme       string `yaml:"scheme"`
	RegisterIPv4 string `yaml:"registerIPv4,omitempty"` // IP that is registered at NRF.
	// IPv6Addr string `yaml:"ipv6Addr,omitempty"`
	BindingIPv4 string `yaml:"bindingIPv4,omitempty"` // IP used to run the server in the node.
	Port        int    `yaml:"port,omitempty"`
	Tls         *Tls   `yaml:"tls,omitempty"`
}

type Tls struct {
	Log string `yaml:"log,omitempty"`

	Pem string `yaml:"pem,omitempty"`

	Key string `yaml:"key,omitempty"`
}

type Nrfclient struct {
	Scheme   string `yaml:"scheme"`
	Ipv4Addr string `yaml:"ipv4Addr,omitempty"`
	Port     int    `yaml:"port,omitempty"`
}

type Udrclient struct {
	Scheme   string `yaml:"scheme"`
	Ipv4Addr string `yaml:"ipv4Addr,omitempty"`
	Port     int    `yaml:"port,omitempty"`
}

type Keys struct {
	UdmProfileAHNPrivateKey string `yaml:"udmProfileAHNPrivateKey,omitempty"`
	UdmProfileAHNPublicKey  string `yaml:"udmProfileAHNPublicKey,omitempty"`
	UdmProfileBHNPrivateKey string `yaml:"udmProfileBHNPrivateKey,omitempty"`
	UdmProfileBHNPublicKey  string `yaml:"udmProfileBHNPublicKey,omitempty"`
}
