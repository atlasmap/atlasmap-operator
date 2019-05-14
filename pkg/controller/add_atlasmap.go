package controller

import (
	"github.com/atlasmap/atlasmap-operator/pkg/controller/atlasmap"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, atlasmap.Add)
}
