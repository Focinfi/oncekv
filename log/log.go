package log

import (
	"os"

	"github.com/Focinfi/oncekv/config"
	"github.com/Sirupsen/logrus"
)

func init() {
	if config.Env().IsProduction() {
		logrus.SetFormatter(&logrus.JSONFormatter{})
		logrus.SetLevel(logrus.WarnLevel)
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{})
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.SetOutput(config.Config().LogOut)

	// Biz
	Biz.Level = logrus.DebugLevel
	Biz.Out = os.Stdout
}

// Internal for internal logic error
var Internal = logrus.New()

// DB for database error logger
var DB = logrus.New()

// Biz for logic logger
var Biz = logrus.New()

// ThirdPartyServiceLogger for service error logger
var ThirdPartyServiceLogger = logrus.New()

// InternalError for logic error
func InternalError(funcName string, message interface{}) {
	Internal.WithFields(logrus.Fields{"function_name": funcName}).Error(message)
}

// DBError log databse error
func DBError(sql interface{}, err error, message interface{}) {
	DB.WithFields(logrus.Fields{"sql": sql, "error": err}).Error(message)
}

// LibError for lib error
func LibError(lib string, message interface{}) {
	DB.WithFields(logrus.Fields{"lib": lib}).Error(message)
}

// ThirdPartyServiceError for third-party service error
func ThirdPartyServiceError(thirdPartyService string, err error, message interface{}, params ...string) {
	DB.WithFields(logrus.Fields{"third_party_service": thirdPartyService, "error": err, "params": params}).Info(message)
}
