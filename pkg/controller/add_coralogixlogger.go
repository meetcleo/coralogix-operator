package controller

import (
	"github.com/coralogix/coralogix-operator/pkg/controller/coralogixlogger"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, coralogixlogger.Add)
}
