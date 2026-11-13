// Package fonts embeds the OFL Liberation faces used by the desktop UI.
package fonts

import "embed"

// Files are the TrueType faces shipped inside the executable.
//
//go:embed LiberationSans-Regular.ttf LiberationSans-Bold.ttf LiberationMono-Regular.ttf LiberationMono-Bold.ttf LICENSE-OFL.txt
var Files embed.FS
