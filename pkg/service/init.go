package service

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/free5gc/udm/internal/context"
	"github.com/free5gc/udm/internal/logger"
	"github.com/free5gc/udm/internal/sbi/consumer"
	"github.com/free5gc/udm/internal/sbi/eventexposure"
	"github.com/free5gc/udm/internal/sbi/httpcallback"
	"github.com/free5gc/udm/internal/sbi/parameterprovision"
	"github.com/free5gc/udm/internal/sbi/subscriberdatamanagement"
	"github.com/free5gc/udm/internal/sbi/ueauthentication"
	"github.com/free5gc/udm/internal/sbi/uecontextmanagement"
	"github.com/free5gc/udm/internal/util"
	"github.com/free5gc/udm/pkg/factory"
	"github.com/free5gc/util/httpwrapper"
	logger_util "github.com/free5gc/util/logger"
)

type UDM struct {
	KeyLogPath string
}

type (
	// Commands information.
	Commands struct {
		config string
	}
)

var commands Commands

var cliCmd = []cli.Flag{
	cli.StringFlag{
		Name:  "config, c",
		Usage: "Load configuration from `FILE`",
	},
	cli.StringFlag{
		Name:  "log, l",
		Usage: "Output NF log to `FILE`",
	},
	cli.StringFlag{
		Name:  "log5gc, lc",
		Usage: "Output free5gc log to `FILE`",
	},
}

func (*UDM) GetCliCmd() (flags []cli.Flag) {
	return cliCmd
}

func (udm *UDM) Initialize(c *cli.Context) error {
	commands = Commands{
		config: c.String("config"),
	}

	if commands.config != "" {
		if err := factory.InitConfigFactory(commands.config); err != nil {
			return err
		}
	} else {
		if err := factory.InitConfigFactory(util.UdmDefaultConfigPath); err != nil {
			return err
		}
	}

	if err := factory.CheckConfigVersion(); err != nil {
		return err
	}

	if _, validErr := factory.UdmConfig.Validate(); validErr != nil {
		return validErr
	}

	udm.SetLogLevel()

	return nil
}

func (udm *UDM) SetLogLevel() {
	if factory.UdmConfig.Logger == nil {
		logger.InitLog.Warnln("UDM config without log level setting!!!")
		return
	}

	if factory.UdmConfig.Logger.UDM != nil {
		if factory.UdmConfig.Logger.UDM.DebugLevel != "" {
			if level, err := logrus.ParseLevel(factory.UdmConfig.Logger.UDM.DebugLevel); err != nil {
				logger.InitLog.Warnf("UDM Log level [%s] is invalid, set to [info] level",
					factory.UdmConfig.Logger.UDM.DebugLevel)
				logger.SetLogLevel(logrus.InfoLevel)
			} else {
				logger.InitLog.Infof("UDM Log level is set to [%s] level", level)
				logger.SetLogLevel(level)
			}
		} else {
			logger.InitLog.Infoln("UDM Log level is default set to [info] level")
			logger.SetLogLevel(logrus.InfoLevel)
		}
		logger.SetReportCaller(factory.UdmConfig.Logger.UDM.ReportCaller)
	}
}

func (udm *UDM) FilterCli(c *cli.Context) (args []string) {
	for _, flag := range udm.GetCliCmd() {
		name := flag.GetName()
		value := fmt.Sprint(c.Generic(name))
		if value == "" {
			continue
		}

		args = append(args, "--"+name, value)
	}
	return args
}

func (udm *UDM) Start() {
	config := factory.UdmConfig
	configuration := config.Configuration
	sbi := configuration.Sbi
	serviceName := configuration.ServiceNameList

	logger.InitLog.Infof("UDM Config Info: Version[%s] Description[%s]", config.Info.Version, config.Info.Description)

	logger.InitLog.Infoln("Server started")

	router := logger_util.NewGinWithLogrus(logger.GinLog)

	eventexposure.AddService(router)
	httpcallback.AddService(router)
	parameterprovision.AddService(router)
	subscriberdatamanagement.AddService(router)
	ueauthentication.AddService(router)
	uecontextmanagement.AddService(router)

	pemPath := util.UdmDefaultPemPath
	keyPath := util.UdmDefaultKeyPath
	if sbi.Tls != nil {
		pemPath = sbi.Tls.Pem
		keyPath = sbi.Tls.Key
	}

	self := context.UDM_Self()
	util.InitUDMContext(self)
	context.UDM_Self().InitNFService(serviceName, config.Info.Version)

	addr := fmt.Sprintf("%s:%d", self.BindingIPv4, self.SBIPort)

	proflie, err := consumer.BuildNFInstance(self)
	if err != nil {
		logger.InitLog.Errorln(err.Error())
	} else {
		var newNrfUri string
		var err1 error
		newNrfUri, self.NfId, err1 = consumer.SendRegisterNFInstance(self.NrfUri, self.NfId, proflie)
		if err1 != nil {
			logger.InitLog.Errorln(err1.Error())
		} else {
			self.NrfUri = newNrfUri
		}
	}

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		defer func() {
			if p := recover(); p != nil {
				// Print stack for panic to log. Fatalf() will let program exit.
				logger.InitLog.Fatalf("panic: %v\n%s", p, string(debug.Stack()))
			}
		}()

		<-signalChannel
		udm.Terminate()
		os.Exit(0)
	}()

	server, err := httpwrapper.NewHttp2Server(addr, udm.KeyLogPath, router)
	if server == nil {
		logger.InitLog.Errorf("Initialize HTTP server failed: %+v", err)
		return
	}

	if err != nil {
		logger.InitLog.Warnf("Initialize HTTP server: +%v", err)
	}

	serverScheme := factory.UdmConfig.Configuration.Sbi.Scheme
	if serverScheme == "http" {
		err = server.ListenAndServe()
	} else if serverScheme == "https" {
		err = server.ListenAndServeTLS(pemPath, keyPath)
	}

	if err != nil {
		logger.InitLog.Fatalf("HTTP server setup failed: %+v", err)
	}
}

func (udm *UDM) Exec(c *cli.Context) error {
	// UDM.Initialize(cfgPath, c)

	logger.InitLog.Traceln("args:", c.String("udmcfg"))
	args := udm.FilterCli(c)
	logger.InitLog.Traceln("filter: ", args)
	command := exec.Command("./udm", args...)

	stdout, err := command.StdoutPipe()
	if err != nil {
		logger.InitLog.Fatalln(err)
	}
	wg := sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer func() {
			if p := recover(); p != nil {
				// Print stack for panic to log. Fatalf() will let program exit.
				logger.InitLog.Fatalf("panic: %v\n%s", p, string(debug.Stack()))
			}
		}()

		in := bufio.NewScanner(stdout)
		for in.Scan() {
			fmt.Println(in.Text())
		}
		wg.Done()
	}()

	stderr, err := command.StderrPipe()
	if err != nil {
		logger.InitLog.Fatalln(err)
	}
	go func() {
		defer func() {
			if p := recover(); p != nil {
				// Print stack for panic to log. Fatalf() will let program exit.
				logger.InitLog.Fatalf("panic: %v\n%s", p, string(debug.Stack()))
			}
		}()

		in := bufio.NewScanner(stderr)
		for in.Scan() {
			fmt.Println(in.Text())
		}
		wg.Done()
	}()

	go func() {
		defer func() {
			if p := recover(); p != nil {
				// Print stack for panic to log. Fatalf() will let program exit.
				logger.InitLog.Fatalf("panic: %v\n%s", p, string(debug.Stack()))
			}
		}()

		if err = command.Start(); err != nil {
			fmt.Printf("UDM Start error: %v", err)
		}
		wg.Done()
	}()

	wg.Wait()

	return err
}

func (udm *UDM) Terminate() {
	logger.InitLog.Infof("Terminating UDM...")
	// deregister with NRF
	problemDetails, err := consumer.SendDeregisterNFInstance()
	if problemDetails != nil {
		logger.InitLog.Errorf("Deregister NF instance Failed Problem[%+v]", problemDetails)
	} else if err != nil {
		logger.InitLog.Errorf("Deregister NF instance Error[%+v]", err)
	} else {
		logger.InitLog.Infof("Deregister from NRF successfully")
	}
	logger.InitLog.Infof("UDM terminated")
}
