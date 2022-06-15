package deployment

import (
	"embed"
)

// This is here to expose the deployment files to code for automated testing.
//go:embed base/*
var Resources embed.FS
