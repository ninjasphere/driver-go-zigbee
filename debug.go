// +build !release

package main

import (
	"github.com/bugsnag/bugsnag-go"
)

var (
	BugsnagKey = "00000000000000000000000000000000" // set by build procedure
)

func init() {
	bugsnag.Configure(bugsnag.Configuration{
		APIKey:       BugsnagKey,
		ReleaseStage: "development",
	})
}
