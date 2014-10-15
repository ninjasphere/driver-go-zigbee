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
		APIKey:       "205838b03710e9d7bf45b3722d7b9ac6",
		ReleaseStage: "development",
	})
}
