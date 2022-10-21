package logger

import (
	"os"
	"time"

	formatter "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"

	logger_util "github.com/free5gc/util/logger"
)

var (
	log         *logrus.Logger
	AppLog      *logrus.Entry
	InitLog     *logrus.Entry
	CfgLog      *logrus.Entry
	Handlelog   *logrus.Entry
	HttpLog     *logrus.Entry
	UeauLog     *logrus.Entry
	UecmLog     *logrus.Entry
	SdmLog      *logrus.Entry
	PpLog       *logrus.Entry
	EeLog       *logrus.Entry
	UtilLog     *logrus.Entry
	SuciLog     *logrus.Entry
	CallbackLog *logrus.Entry
	ContextLog  *logrus.Entry
	ConsumerLog *logrus.Entry
	GinLog      *logrus.Entry
)

func init() {
	log = logrus.New()
	log.SetReportCaller(false)

	log.Formatter = &formatter.Formatter{
		TimestampFormat: time.RFC3339Nano,
		TrimMessages:    true,
		NoFieldsSpace:   true,
		HideKeys:        true,
		FieldsOrder:     []string{"component", "category"},
	}

	AppLog = log.WithFields(logrus.Fields{"component": "UDM", "category": "App"})
	InitLog = log.WithFields(logrus.Fields{"component": "UDM", "category": "Init"})
	CfgLog = log.WithFields(logrus.Fields{"component": "UDM", "category": "CFG"})
	Handlelog = log.WithFields(logrus.Fields{"component": "UDM", "category": "HDLR"})
	HttpLog = log.WithFields(logrus.Fields{"component": "UDM", "category": "HTTP"})
	UeauLog = log.WithFields(logrus.Fields{"component": "UDM", "category": "UEAU"})
	UecmLog = log.WithFields(logrus.Fields{"component": "UDM", "category": "UECM"})
	SdmLog = log.WithFields(logrus.Fields{"component": "UDM", "category": "SDM"})
	PpLog = log.WithFields(logrus.Fields{"component": "UDM", "category": "PP"})
	EeLog = log.WithFields(logrus.Fields{"component": "UDM", "category": "EE"})
	UtilLog = log.WithFields(logrus.Fields{"component": "UDM", "category": "Util"})
	SuciLog = log.WithFields(logrus.Fields{"component": "UDM", "category": "Suci"})
	CallbackLog = log.WithFields(logrus.Fields{"component": "UDM", "category": "CB"})
	ContextLog = log.WithFields(logrus.Fields{"component": "UDM", "category": "CTX"})
	ConsumerLog = log.WithFields(logrus.Fields{"component": "UDM", "category": "Consumer"})
	GinLog = log.WithFields(logrus.Fields{"component": "UDM", "category": "GIN"})
}

func LogFileHook(logNfPath string, log5gcPath string) error {
	if fullPath, err := logger_util.CreateFree5gcLogFile(log5gcPath); err == nil {
		if fullPath != "" {
			free5gcLogHook, hookErr := logger_util.NewFileHook(fullPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o666)
			if hookErr != nil {
				return hookErr
			}
			log.Hooks.Add(free5gcLogHook)
		}
	} else {
		return err
	}

	if fullPath, err := logger_util.CreateNfLogFile(logNfPath, "udm.log"); err == nil {
		selfLogHook, hookErr := logger_util.NewFileHook(fullPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o666)
		if hookErr != nil {
			return hookErr
		}
		log.Hooks.Add(selfLogHook)
	} else {
		return err
	}

	return nil
}

func SetLogLevel(level logrus.Level) {
	log.SetLevel(level)
}

func SetReportCaller(enable bool) {
	log.SetReportCaller(enable)
}
