package service

import (
  
	"context"
	"io"
	"os"
	"runtime/debug"
	"sync"

	"github.com/sirupsen/logrus"

	udm_context "github.com/free5gc/udm/internal/context"
	"github.com/free5gc/udm/internal/logger"
	"github.com/free5gc/udm/internal/sbi"
	"github.com/free5gc/udm/internal/sbi/consumer"
	"github.com/free5gc/udm/internal/sbi/processor"
	"github.com/free5gc/udm/pkg/app"
	"github.com/free5gc/udm/pkg/factory"
)

var UDM *UdmApp

var _ app.App = &UdmApp{}

type UdmApp struct {
	app.App
	consumer.ConsumerUdm
	processor.ProcessorUdm
	sbi.ServerUdm

	udmCtx *udm_context.UDMContext
	cfg    *factory.Config

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	sbiServer *sbi.Server
	consumer  *consumer.Consumer
	processor *processor.Processor
}

func GetApp() *UdmApp {
	return UDM
}

func NewApp(ctx context.Context, cfg *factory.Config, tlsKeyLogPath string) (*UdmApp, error) {
	udm := &UdmApp{
		cfg: cfg,
		wg:  sync.WaitGroup{},
	}
	udm.SetLogEnable(cfg.GetLogEnable())
	udm.SetLogLevel(cfg.GetLogLevel())
	udm.SetReportCaller(cfg.GetLogReportCaller())

	processor, err_p := processor.NewProcessor(udm)
	if err_p != nil {
		return udm, err_p
	}
	udm.processor = processor

	consumer, err := consumer.NewConsumer(udm)
	if err != nil {
		return udm, err
	}
	udm.consumer = consumer

	udm.ctx, udm.cancel = context.WithCancel(ctx)
	udm.udmCtx = udm_context.GetSelf()

	if udm.sbiServer, err = sbi.NewServer(udm, tlsKeyLogPath); err != nil {
		return nil, err
	}

	UDM = udm

	return udm, nil
}

func (a *UdmApp) SetLogEnable(enable bool) {
	logger.MainLog.Infof("Log enable is set to [%v]", enable)
	if enable && logger.Log.Out == os.Stderr {
		return
	} else if !enable && logger.Log.Out == io.Discard {
		return
	}

	a.cfg.SetLogEnable(enable)
	if enable {
		logger.Log.SetOutput(os.Stderr)
	} else {
		logger.Log.SetOutput(io.Discard)
	}
}

func (a *UdmApp) SetLogLevel(level string) {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		logger.MainLog.Warnf("Log level [%s] is invalid", level)
		return
	}

	logger.MainLog.Infof("Log level is set to [%s]", level)
	if lvl == logger.Log.GetLevel() {
		return
	}

	a.cfg.SetLogLevel(level)
	logger.Log.SetLevel(lvl)
}

func (a *UdmApp) SetReportCaller(reportCaller bool) {
	logger.MainLog.Infof("Report Caller is set to [%v]", reportCaller)
	if reportCaller == logger.Log.ReportCaller {
		return
	}
	a.cfg.SetLogReportCaller(reportCaller)
	logger.Log.SetReportCaller(reportCaller)
}

func (a *UdmApp) Start() {
	logger.InitLog.Infoln("Server started")

	a.wg.Add(1)
	go a.listenShutdownEvent()

	if err := a.sbiServer.Run(context.Background(), &a.wg); err != nil {
		logger.MainLog.Fatalf("Run SBI server failed: %+v", err)
	}
	/*config := factory.UdmConfig
	configuration := config.Configuration
	sbi := configuration.Sbi

	logger.InitLog.Infof("UDM Config Info: Version[%s] Description[%s]", config.Info.Version, config.Info.Description)

	logger.InitLog.Infoln("Server started")

	router := logger_util.NewGinWithLogrus(logger.GinLog)

	eventexposure.AddService(router)
	httpcallback.AddService(router)
	parameterprovision.AddService(router)
	subscriberdatamanagement.AddService(router)
	ueauthentication.AddService(router)
	uecontextmanagement.AddService(router)

	pemPath := factory.UdmDefaultCertPemPath
	keyPath := factory.UdmDefaultPrivateKeyPath
	if sbi.Tls != nil {
		pemPath = sbi.Tls.Pem
		keyPath = sbi.Tls.Key
	}

	self := a.udmCtx
	udm_context.InitUdmContext(self)

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
		a.Terminate()
		os.Exit(0)
	}()

	server, err := httpwrapper.NewHttp2Server(addr, tlsKeyLogPath, router)
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
	}*/
}

func (a *UdmApp) Terminate() {
	logger.MainLog.Infof("Terminating UDM...")
	a.cancel()
	a.CallServerStop()

	// deregister with NRF
	problemDetails, err := a.Consumer().SendDeregisterNFInstance()
	if problemDetails != nil {
		logger.MainLog.Errorf("Deregister NF instance Failed Problem[%+v]", problemDetails)
	} else if err != nil {
		logger.MainLog.Errorf("Deregister NF instance Error[%+v]", err)
	} else {
		logger.MainLog.Infof("Deregister from NRF successfully")
	}
	logger.MainLog.Infof("CHF SBI Server terminated")
}

func (a *UdmApp) listenShutdownEvent() {
	defer func() {
		if p := recover(); p != nil {
			// Print stack for panic to log. Fatalf() will let program exit.
			logger.MainLog.Fatalf("panic: %v\n%s", p, string(debug.Stack()))
		}
		a.wg.Done()
	}()

	<-a.ctx.Done()
	a.Terminate()
}

func (a *UdmApp) CallServerStop() {
	if a.sbiServer != nil {
		a.sbiServer.Stop()
	}
}

func (a *UdmApp) WaitRoutineStopped() {
	a.wg.Wait()
	logger.MainLog.Infof("CHF App is terminated")
}

func (a *UdmApp) Config() *factory.Config {
	return a.cfg
}

func (a *UdmApp) Context() *udm_context.UDMContext {
	return a.udmCtx
}

func (a *UdmApp) CancelContext() context.Context {
	return a.ctx
}

func (a *UdmApp) Consumer() *consumer.Consumer {
	return a.consumer
}

func (a *UdmApp) Processor() *processor.Processor {
	return a.processor
}
