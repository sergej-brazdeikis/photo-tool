package webshare

import "embed"

// FS holds the read-only share page template and stylesheet (Story 3.3).
//
//go:embed share.html share_package.html share.css
var FS embed.FS
