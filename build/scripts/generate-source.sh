#!/bin/bash

ATLASMAP_IMAGE=$1
ATLASMAP_IMAGE_TAG=$2

cat << EOF > pkg/config/config.go
package config

// *************************************
// THIS FILE IS GENERATED - DO NOT EDIT
// *************************************

import "github.com/atlasmap/atlasmap-operator/pkg/util"

// AtlasMapConfig --
type AtlasMapConfig struct {
	AtlasMapImage string
	Version       string
}

// DefaultConfiguration --
var DefaultConfiguration = AtlasMapConfig{
	AtlasMapImage: "${ATLASMAP_IMAGE}",
	Version:       "${ATLASMAP_IMAGE_TAG}",
}

func (c *AtlasMapConfig) GetAtlasMapImage() string {
  return util.ImageName(c.AtlasMapImage, c.Version)
}
EOF

gofmt -w pkg/config/config.go
