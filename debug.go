// +build !release

package main

import (
	"github.com/bugsnag/bugsnag-go"
	"github.com/juju/loggo"
	"github.com/ninjasphere/go-ninja/logger"
)

var debug = logger.GetLogger("").Warningf

func init() {
	logger.GetLogger("").SetLogLevel(loggo.DEBUG)

	bugsnag.Configure(bugsnag.Configuration{
		APIKey:       "6145ec8c75d22e6236758aca31247a63",
		ReleaseStage: "development",
	})
}
