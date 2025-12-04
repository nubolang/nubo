package config

import (
	"embed"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed base.yaml
var baseConfigFile embed.FS

// Config represents the configuration for the application.
type Config struct {
	Syntax struct {
		Lexer struct {
			Debug struct {
				Enable string `yaml:"enable"`
				File   string `yaml:"file"`
			} `yaml:"debug"`
		} `yaml:"lexer"`
		Tokenizer struct {
			Context struct {
				Deadline int `yaml:"deadline"`
			} `yaml:"context"`
			Debug struct {
				Enable string `yaml:"enable"`
				File   string `yaml:"file"`
			} `yaml:"debug"`
		} `yaml:"tokenizer"`
	} `yaml:"syntax"`

	Runtime struct {
		Server struct {
			Address           string `yaml:"address"`
			MaxConcurrency    int    `yaml:"max_concurrency"`
			MaxUploadSizeByte int64  `yaml:"max_upload_size_byte"`
			MaxUploadFileSize int64  `yaml:"max_upload_file_size"`
		} `yaml:"server"`
		Std struct {
			Allow    string `yaml:"allow"`
			Disallow string `yaml:"disallow"`
		} `yaml:"std"`
		Events struct {
			Enabled            bool `yaml:"enabled"`
			MaxWorkersPerTopic int  `yaml:"max_workers_per_topic"`
			ChannelBufferSize  int  `yaml:"channel_buffer_size"`
		} `yaml:"events"`
		Interpreter struct {
			Import struct {
				Prefix map[string]string `yaml:"prefix"`
			} `yaml:"import"`
		} `yaml:"interpreter"`
	} `yaml:"runtime"`

	Logging struct {
		Level   string `yaml:"level"`
		Loggers struct {
			Console struct {
				Use      bool   `yaml:"use"`
				Encoding string `yaml:"encoding"`
			} `yaml:"console"`
			File struct {
				Use      bool   `yaml:"use"`
				Path     string `yaml:"path"`
				Encoding string `yaml:"encoding"`
			} `yaml:"file"`
		} `yaml:"loggers"`
	} `yaml:"logging"`
}

// ApplyDefaults fills missing values with defaults
func (c *Config) ApplyDefaults() {
	// defaults
	if c.Syntax.Lexer.Debug.Enable == "" || !slices.Contains([]string{"development", "production", "any"}, c.Syntax.Lexer.Debug.Enable) {
		c.Syntax.Lexer.Debug.Enable = "development"
	}
	if c.Syntax.Lexer.Debug.File == "" {
		c.Syntax.Lexer.Debug.File = "{nubo_dir}/debug/lexer.yaml"
	}

	if c.Syntax.Tokenizer.Context.Deadline == 0 {
		c.Syntax.Tokenizer.Context.Deadline = 5000
	}
	if c.Syntax.Tokenizer.Debug.Enable == "" || !slices.Contains([]string{"development", "production", "any"}, c.Syntax.Tokenizer.Debug.Enable) {
		c.Syntax.Tokenizer.Debug.Enable = "development"
	}
	if c.Syntax.Tokenizer.Debug.File == "" {
		c.Syntax.Tokenizer.Debug.File = "{nubo_dir}/debug/ast.yaml"
	}

	// server defaults
	if c.Runtime.Server.Address == "" {
		c.Runtime.Server.Address = ":3000"
	}
	if c.Runtime.Server.MaxConcurrency == 0 {
		c.Runtime.Server.MaxConcurrency = 10
	}
	if c.Runtime.Server.MaxUploadSizeByte == 0 {
		c.Runtime.Server.MaxUploadSizeByte = 1_000_000
	}
	if c.Runtime.Server.MaxUploadFileSize == 0 {
		c.Runtime.Server.MaxUploadFileSize = 5 << 20
	} else {
		c.Runtime.Server.MaxUploadFileSize = c.Runtime.Server.MaxUploadFileSize << 20
	}

	// std defaults
	if c.Runtime.Std.Allow == "" {
		c.Runtime.Std.Allow = ":all"
	}
	if c.Runtime.Std.Disallow == "" {
		c.Runtime.Std.Disallow = "-"
	}

	// events defaults
	if c.Runtime.Events.MaxWorkersPerTopic == 0 {
		c.Runtime.Events.MaxWorkersPerTopic = 10
	}
	if c.Runtime.Events.ChannelBufferSize == 0 {
		c.Runtime.Events.ChannelBufferSize = 1024
	}

	// interpreter defaults
	if c.Runtime.Interpreter.Import.Prefix == nil {
		c.Runtime.Interpreter.Import.Prefix = map[string]string{"~": "{current_dir}"}
	}

	// logging defaults
	if c.Logging.Level == "" {
		c.Logging.Level = "prod"
	}
	c.Logging.Level = strings.ToUpper(c.Logging.Level)
	if c.Logging.Loggers.Console.Encoding == "" {
		c.Logging.Loggers.Console.Encoding = "console"
	}
	if c.Logging.Loggers.File.Path == "" {
		c.Logging.Loggers.File.Path = "{current_dir}/logs/nubo.log"
	}
	if c.Logging.Loggers.File.Encoding == "" {
		c.Logging.Loggers.File.Encoding = "json"
	}
}

func (c *Config) String() string {
	if b, err := yaml.Marshal(c); err != nil {
		return err.Error()
	} else {
		return string(b)
	}
}

// ReplaceVariables substitutes placeholders in strings using a map
func ReplaceVariables(input string, vars map[string]string) string {
	for k, v := range vars {
		input = strings.ReplaceAll(input, "{"+k+"}", v)
	}
	return input
}

var (
	Current *Config
	Base    string
	Nubo    string
)

func Verify() {
	if Current == nil {
		cfg := &Config{}
		cfg.ApplyDefaults()
		Current = cfg
	}
}

func Load() error {
	base, err := BaseDir()
	if err != nil {
		return err
	}

	Base = base

	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	Nubo = filepath.Join(pwd, ".nubo")

	config := filepath.Join(base, "config.yaml")
	if _, err := os.Stat(config); err != nil {
		if err := createConfigFile(config); err != nil {
			return err
		}
	}

	file, err := os.Open(config)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}

	cfg.ApplyDefaults()

	// Example: replace {nubo_dir} in lexer file
	vars := map[string]string{
		"nubo_dir":    Nubo,
		"current_dir": pwd,
		"date":        time.Now().Format("2006-01-02"),
		"datedir":     time.Now().Format("2006/01/02"),
		"time":        time.Now().Format("15-04-05"),
	}
	cfg.Syntax.Lexer.Debug.File = ReplaceVariables(cfg.Syntax.Lexer.Debug.File, vars)
	cfg.Syntax.Tokenizer.Debug.File = ReplaceVariables(cfg.Syntax.Tokenizer.Debug.File, vars)
	for key, value := range cfg.Runtime.Interpreter.Import.Prefix {
		cfg.Runtime.Interpreter.Import.Prefix[key] = ReplaceVariables(value, vars)
	}
	cfg.Logging.Loggers.File.Path = ReplaceVariables(cfg.Logging.Loggers.File.Path, vars)

	Current = &cfg
	return nil
}

func createConfigFile(path string) error {
	baseFile, err := baseConfigFile.Open("base.yaml")
	if err != nil {
		return err
	}
	defer baseFile.Close()

	configFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer configFile.Close()

	_, err = io.Copy(configFile, baseFile)
	return err
}
