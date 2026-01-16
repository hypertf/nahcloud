package static

import _ "embed"

//go:embed output.css
var CSS string

//go:embed logo.png
var Logo []byte
