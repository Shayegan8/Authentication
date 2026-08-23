package myblog

import (
	_ "embed"
)

//go:embed config.json
var ConfigBuffer []byte

var Config map[string]string
