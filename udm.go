package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"gofree5gc/src/app"
	// "gofree5gc/lib/milenage"
	// m "gofree5gc/lib/openapi/models"
	"gofree5gc/src/udm/logger"
	"gofree5gc/src/udm/udm_service"
	"gofree5gc/src/udm/version"
	"os"
)

var UDM = &udm_service.UDM{}

var appLog *logrus.Entry

func init() {
	appLog = logger.AppLog
}

func main() {
	app := cli.NewApp()
	app.Name = "udm"
	fmt.Print(app.Name, "\n")
	appLog.Infoln("UDM version: ", version.GetVersion())
	app.Usage = "-free5gccfg common configuration file -udmcfg udm configuration file"
	app.Action = action
	app.Flags = UDM.GetCliCmd()
	if err := app.Run(os.Args); err != nil {
		fmt.Printf("UDM Run error: %v", err)
	}

	// appLog.Infoln(app.Name)

}

func action(c *cli.Context) {
	app.AppInitializeWillInitialize(c.String("free5gccfg"))
	UDM.Initialize(c)
	UDM.Start()
}
