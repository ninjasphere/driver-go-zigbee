// +build release

package main

import (
	"github.com/bugsnag/bugsnag-go"
	"github.com/juju/loggo"
	"github.com/ninjasphere/go-ninja/logger"
)

func init() {
	logger.GetLogger("").SetLogLevel(loggo.INFO)

	bugsnag.Configure(bugsnag.Configuration{
		APIKey:       "205838b03710e9d7bf45b3722d7b9ac6",
		ReleaseStage: "production",
	})
}
