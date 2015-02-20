// +build release

package main

import (
	"github.com/bugsnag/bugsnag-go"
)

var (
	BugsnagKey = "" // set by build procedure
)

func init() {
	bugsnag.Configure(bugsnag.Configuration{
		APIKey:       BugsnagKey,
		ReleaseStage: "production",
	})
}
