package hex

import (
	_ "embed"
	"encoding/hex"
)

//go:embed package.nubo
var Source string

func Functions() map[string]any {
	return map[string]any{
		"encode": encode,
		"decode": decode,
	}
}

func encode(value string) string {
	return hex.EncodeToString([]byte(value))
}

func decode(value string) (string, error) {
	data, err := hex.DecodeString(value)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
