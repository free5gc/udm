/*
 * UDM Configuration Factory
 */

package factory

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"

	"github.com/free5gc/udm/internal/logger"
)

var UdmConfig Config

// TODO: Support configuration update from REST api
func InitConfigFactory(f string) error {
	if content, err := ioutil.ReadFile(f); err != nil {
		return err
	} else {
		UdmConfig = Config{}

		if yamlErr := yaml.Unmarshal(content, &UdmConfig); yamlErr != nil {
			return yamlErr
		}
	}

	return nil
}

func CheckConfigVersion() error {
	currentVersion := UdmConfig.GetVersion()

	if currentVersion != UdmExpectedConfigVersion {
		return fmt.Errorf("config version is [%s], but expected is [%s].",
			currentVersion, UdmExpectedConfigVersion)
	}

	logger.CfgLog.Infof("config version [%s]", currentVersion)

	return nil
}
