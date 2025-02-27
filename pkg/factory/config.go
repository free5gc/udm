/*
 * UDM Configuration Factory
 */

package factory

import (
	"fmt"
	"sync"

	"github.com/asaskevich/govalidator"
	"github.com/free5gc/udm/internal/logger"
	"github.com/free5gc/udm/pkg/suci"
)

const (
	UdmDefaultTLSKeyLogPath       = "./log/udmsslkey.log"
	UdmDefaultCertPemPath         = "./cert/udm.pem"
	UdmDefaultPrivateKeyPath      = "./cert/udm.key"
	UdmDefaultConfigPath          = "./config/udmcfg.yaml"
	UdmSbiDefaultIP               = "127.0.0.3"
	UdmSbiDefaultPort             = 8000
	UdmSbiDefaultScheme           = "https"
	UdmDefaultNrfUri              = "https://127.0.0.10:8000"
	UdmSorprotectionResUriPrefix  = "/nudm-sorprotection/v1"
	UdmAuthResUriPrefix           = "/nudm-auth/v1"
	UdmfUpuprotectionResUriPrefix = "/nudm-upuprotection/v1"
	UdmEcmResUriPrefix            = "/nudm-ecm/v1"
	UdmSdmResUriPrefix            = "/nudm-sdm/v2"
	UdmEeResUriPrefix             = "/nudm-ee/v1"
	UdmDrResUriPrefix             = "/nudr-dr/v1"
	UdmUecmResUriPrefix           = "/nudm-uecm/v1"
	UdmPpResUriPrefix             = "/nudm-pp/v1"
	UdmUeauResUriPrefix           = "/nudm-ueau/v1"
	UdmMtResUrdPrefix             = "/nudm-mt/v1"
	UdmNiddauResUriPrefix         = "/nudm-niddau/v1"
	UdmRsdsResUriPrefix           = "/nudm-rsds/v1"
	UdmSsauResUriPrefix           = "/nudm-ssau/v1"
	UdmUeidResUriPrefix           = "/nudm-ueid/v1"
)

type Config struct {
	Info          *Info          `yaml:"info" valid:"required"`
	Configuration *Configuration `yaml:"configuration" valid:"required"`
	Logger        *Logger        `yaml:"logger" valid:"required"`
	sync.RWMutex
}

func (c *Config) Validate() (bool, error) {
	if configuration := c.Configuration; configuration != nil {
		if result, err := configuration.validate(); err != nil {
			return result, err
		}
	}

	result, err := govalidator.ValidateStruct(c)
	return result, appendInvalid(err)
}

type Info struct {
	Version     string `yaml:"version,omitempty" valid:"required,in(1.0.4)"`
	Description string `yaml:"description,omitempty" valid:"type(string)"`
}

type Configuration struct {
	Sbi             *Sbi               `yaml:"sbi" valid:"optional"`
	ServiceNameList []string           `yaml:"serviceNameList,omitempty"  valid:"required"`
	NrfUri          string             `yaml:"nrfUri,omitempty"  valid:"required, url"`
	NrfCertPem      string             `yaml:"nrfCertPem,omitempty" valid:"optional"`
	SuciProfiles    []suci.SuciProfile `yaml:"SuciProfile,omitempty"`
}
type Logger struct {
	Enable       bool   `yaml:"enable" valid:"type(bool)"`
	Level        string `yaml:"level" valid:"required,in(trace|debug|info|warn|error|fatal|panic)"`
	ReportCaller bool   `yaml:"reportCaller" valid:"type(bool)"`
}

func (c *Configuration) validate() (bool, error) {
	if sbi := c.Sbi; sbi != nil {
		if result, err := sbi.validate(); err != nil {
			return result, err
		}
	} else {
		sbi := Sbi{}
		result, err := sbi.validate()
		if err != nil {
			return result, err
		} else {
			c.Sbi = &sbi
		}
	}

	if c.ServiceNameList != nil {
		var errs govalidator.Errors
		for _, v := range c.ServiceNameList {
			if v != "nudm-sdm" && v != "nudm-uecm" && v != "nudm-ueau" && v != "nudm-ee" && v != "nudm-pp" {
				err := fmt.Errorf("Invalid ServiceNameList: [%s],"+
					" value should be nudm-sdm or nudm-uecm or nudm-ueau or nudm-ee or nudm-pp", v)
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return false, error(errs)
		}
	}

	if c.SuciProfiles != nil {
		var errs govalidator.Errors
		for _, s := range c.SuciProfiles {
			protectScheme := s.ProtectionScheme
			if result := govalidator.StringMatches(protectScheme, "^[A-F0-9]{1}$"); !result {
				err := fmt.Errorf("Invalid ProtectionScheme: %s, should be a single hexadecimal digit", protectScheme)
				errs = append(errs, err)
			}

			privateKey := s.PrivateKey
			if result := govalidator.StringMatches(privateKey, "^[A-Fa-f0-9]{64}$"); !result {
				err := fmt.Errorf("Invalid PrivateKey: %s, should be 64 hexadecimal digits", privateKey)
				errs = append(errs, err)
			}

			publicKey := s.PublicKey
			if result := govalidator.StringMatches(publicKey, "^[A-Fa-f0-9]{64,130}$"); !result {
				err := fmt.Errorf("Invalid PublicKey: %s, should be 64(profile A), 66(profile B, compressed),"+
					"or 130(profile B, uncompressed) hexadecimal digits", publicKey)
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return false, error(errs)
		}
	}

	result, err := govalidator.ValidateStruct(c)
	return result, err
}

func (c *Config) GetCertPemPath() string {
	c.RLock()
	defer c.RUnlock()
	return c.Configuration.Sbi.Tls.Pem
}

func (c *Config) GetCertKeyPath() string {
	c.RLock()
	defer c.RUnlock()
	return c.Configuration.Sbi.Tls.Key
}

type Sbi struct {
	Scheme       string `yaml:"scheme" valid:"in(http|https),optional"`
	RegisterIPv4 string `yaml:"registerIPv4,omitempty" valid:"host,optional"` // IP that is registered at NRF.
	RegisterIP   string `yaml:"registerIP,omitempty" valid:"host,optional"`   // IP that is registered at NRF.
	BindingIPv4  string `yaml:"bindingIPv4,omitempty" valid:"host,optional"`  // IP used to run the server in the node.
	BindingIP    string `yaml:"bindingIP,omitempty" valid:"host,optional"`    // IP used to run the server in the node.
	Port         int    `yaml:"port,omitempty" valid:"port,optional"`
	Tls          *Tls   `yaml:"tls,omitempty" valid:"optional"`
}

func (s *Sbi) validate() (bool, error) {
	// Set a default Schme if the Configuration does not provides one
	if s.Scheme == "" {
		s.Scheme = UdmSbiDefaultScheme
	}

	// Set BindingIP/RegisterIP from deprecated BindingIPv4/RegisterIPv4
	if s.BindingIP == "" && s.BindingIPv4 != "" {
		s.BindingIP = s.BindingIPv4
	}
	if s.RegisterIP == "" && s.RegisterIPv4 != "" {
		s.RegisterIP = s.RegisterIPv4
	}

	// Set a default BindingIP/RegisterIP if the Configuration does not provides them
	if s.BindingIP == "" && s.RegisterIP == "" {
		s.BindingIP = UdmSbiDefaultIP
		s.RegisterIP = UdmSbiDefaultIP
	} else {
		// Complete any missing BindingIP/RegisterIP from RegisterIP/BindingIP
		if s.BindingIP == "" {
			s.BindingIP = s.RegisterIP
		} else if s.RegisterIP == "" {
			s.RegisterIP = s.BindingIP
		}
	}

	// Set a default Port if the Configuration does not provides one
	if s.Port == 0 {
		s.Port = UdmSbiDefaultPort
	}

	if tls := s.Tls; tls != nil {
		if result, err := tls.validate(); err != nil {
			return result, err
		}
	}

	result, err := govalidator.ValidateStruct(s)
	return result, err
}

type Tls struct {
	Pem string `yaml:"pem,omitempty" valid:"type(string),minstringlength(1),required"`
	Key string `yaml:"key,omitempty" valid:"type(string),minstringlength(1),required"`
}

func (t *Tls) validate() (bool, error) {
	result, err := govalidator.ValidateStruct(t)
	return result, err
}

func appendInvalid(err error) error {
	var errs govalidator.Errors

	if err == nil {
		return nil
	}

	es := err.(govalidator.Errors).Errors()
	for _, e := range es {
		errs = append(errs, fmt.Errorf("Invalid %w", e))
	}

	return error(errs)
}

func (c *Config) GetVersion() string {
	c.RLock()
	defer c.RUnlock()

	if c.Info.Version != "" {
		return c.Info.Version
	}
	return ""
}

func (c *Config) SetLogEnable(enable bool) {
	c.Lock()
	defer c.Unlock()

	if c.Logger == nil {
		logger.CfgLog.Warnf("Logger should not be nil")
		c.Logger = &Logger{
			Enable: enable,
			Level:  "info",
		}
	} else {
		c.Logger.Enable = enable
	}
}

func (c *Config) SetLogLevel(level string) {
	c.Lock()
	defer c.Unlock()

	if c.Logger == nil {
		logger.CfgLog.Warnf("Logger should not be nil")
		c.Logger = &Logger{
			Level: level,
		}
	} else {
		c.Logger.Level = level
	}
}

func (c *Config) SetLogReportCaller(reportCaller bool) {
	c.Lock()
	defer c.Unlock()

	if c.Logger == nil {
		logger.CfgLog.Warnf("Logger should not be nil")
		c.Logger = &Logger{
			Level:        "info",
			ReportCaller: reportCaller,
		}
	} else {
		c.Logger.ReportCaller = reportCaller
	}
}

func (c *Config) GetLogEnable() bool {
	c.RLock()
	defer c.RUnlock()
	if c.Logger == nil {
		logger.CfgLog.Warnf("Logger should not be nil")
		return false
	}
	return c.Logger.Enable
}

func (c *Config) GetLogLevel() string {
	c.RLock()
	defer c.RUnlock()
	if c.Logger == nil {
		logger.CfgLog.Warnf("Logger should not be nil")
		return "info"
	}
	return c.Logger.Level
}

func (c *Config) GetLogReportCaller() bool {
	c.RLock()
	defer c.RUnlock()
	if c.Logger == nil {
		logger.CfgLog.Warnf("Logger should not be nil")
		return false
	}
	return c.Logger.ReportCaller
}

func (c *Config) GetSbiPort() int {
	c.RLock()
	defer c.RUnlock()
	if c.Configuration != nil && c.Configuration.Sbi != nil && c.Configuration.Sbi.Port != 0 {
		return c.Configuration.Sbi.Port
	}
	return UdmSbiDefaultPort
}

func (c *Config) GetSbiScheme() string {
	c.RLock()
	defer c.RUnlock()
	if c.Configuration != nil && c.Configuration.Sbi != nil && c.Configuration.Sbi.Scheme != "" {
		return c.Configuration.Sbi.Scheme
	}
	return UdmSbiDefaultScheme
}
