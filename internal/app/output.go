package app

import (
	"encoding/json"
	"io"

	"gopkg.in/yaml.v3"

	"github.com/nicerobot/tools.admin/internal/constants"
)

type (
	format string // format represents an output format.
)

const (
	formatJSON format = "json" // formatJSON represents JSON output format.
	formatYAML format = "yaml" // formatYAML represents YAML output format.
)

// output writes data to the given writer in the specified format
func output(writer io.Writer, format format, data any) error {
	if format == "" {
		format = formatJSON // Default to JSON
	}
	switch format {
	case formatJSON:
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)
		return encoder.Encode(data)
	case formatYAML:
		encoder := yaml.NewEncoder(writer)
		defer encoder.Close()
		return encoder.Encode(data)
	default:
		return constants.ErrInvalidValue.With(nil, "format", string(format))
	}
}
