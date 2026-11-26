package logger

import (
	"strings"

	"github.com/sirupsen/logrus"

	logger_util "github.com/free5gc/util/logger"
)

var (
	Log         *logrus.Logger
	NfLog       *logrus.Entry
	MainLog     *logrus.Entry
	InitLog     *logrus.Entry
	CfgLog      *logrus.Entry
	CtxLog      *logrus.Entry
	GinLog      *logrus.Entry
	SBILog      *logrus.Entry
	ConsumerLog *logrus.Entry
	HttpLog     *logrus.Entry
	UeauLog     *logrus.Entry
	UecmLog     *logrus.Entry
	SdmLog      *logrus.Entry
	PpLog       *logrus.Entry
	EeLog       *logrus.Entry
	UtilLog     *logrus.Entry
	SuciLog     *logrus.Entry
	CallbackLog *logrus.Entry
	ProcLog     *logrus.Entry
)

// ginLogSanitizer escapes line breaks and terminal control characters in
// UDM Gin access log messages, so request-derived values (path, user, error)
// cannot forge additional log records. It is local to UDM, so the shared
// free5gc/util module does not need to change.
type ginLogSanitizer struct{}

var ginLogEscaper = strings.NewReplacer(
	"\r", `\r`,
	"\n", `\n`,
	"\x1b", `\x1b`,
	"\u2028", `\u2028`,
	"\u2029", `\u2029`,
)

func (ginLogSanitizer) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (ginLogSanitizer) Fire(entry *logrus.Entry) error {
	if entry.Data[logger_util.FieldCategory] != "GIN" {
		return nil
	}

	entry.Message = ginLogEscaper.Replace(entry.Message)
	return nil
}

func init() {
	fieldsOrder := []string{
		logger_util.FieldNF,
		logger_util.FieldCategory,
	}

	Log = logger_util.New(fieldsOrder)
	Log.AddHook(ginLogSanitizer{})
	NfLog = Log.WithField(logger_util.FieldNF, "UDM")
	MainLog = NfLog.WithField(logger_util.FieldCategory, "Main")
	InitLog = NfLog.WithField(logger_util.FieldCategory, "Init")
	CfgLog = NfLog.WithField(logger_util.FieldCategory, "CFG")
	CtxLog = NfLog.WithField(logger_util.FieldCategory, "CTX")
	GinLog = NfLog.WithField(logger_util.FieldCategory, "GIN")
	SBILog = NfLog.WithField(logger_util.FieldCategory, "SBI")
	ConsumerLog = NfLog.WithField(logger_util.FieldCategory, "Consumer")
	ProcLog = NfLog.WithField(logger_util.FieldCategory, "Proc")
	HttpLog = NfLog.WithField(logger_util.FieldCategory, "HTTP")
	UeauLog = NfLog.WithField(logger_util.FieldCategory, "UEAU")
	UecmLog = NfLog.WithField(logger_util.FieldCategory, "UECM")
	SdmLog = NfLog.WithField(logger_util.FieldCategory, "SDM")
	PpLog = NfLog.WithField(logger_util.FieldCategory, "PP")
	EeLog = NfLog.WithField(logger_util.FieldCategory, "EE")
	UtilLog = NfLog.WithField(logger_util.FieldCategory, "Util")
	SuciLog = NfLog.WithField(logger_util.FieldCategory, "Suci")
	CallbackLog = NfLog.WithField(logger_util.FieldCategory, "Callback")
}
