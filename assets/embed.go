// Package assets embeds the recovered Indianapolis 500 runtime artwork.
package assets

import "embed"

// Original contains assets reconstructed from the supported Amiga disk.
//
//go:embed original/*
var Original embed.FS
