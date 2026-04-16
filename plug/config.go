package plug

// PlugConfig maps _plug.yaml.
type PlugConfig struct {
	Plugin PluginMeta `yaml:"plugin"`
}

// PluginMeta holds plugin build settings.
type PluginMeta struct {
	CGO          bool            `yaml:"cgo"`
	Source       string          `yaml:"source"`
	Cmd          string          `yaml:"cmd"`
	Binary       string          `yaml:"binary"`
	Architecture []string        `yaml:"architecture"`
	Transport    TransportConfig `yaml:"transport"`
}

// TransportConfig describes the communication mode between host and plugin.
type TransportConfig struct {
	// Mode is "stdio" or "tcp".
	Mode string `yaml:"mode"`
}
