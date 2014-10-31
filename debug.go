// +build !release

package main

import (
	"github.com/bugsnag/bugsnag-go"
)

func init() {
	bugsnag.Configure(bugsnag.Configuration{
		APIKey:       "6145ec8c75d22e6236758aca31247a63",
		ReleaseStage: "development",
	})
}
