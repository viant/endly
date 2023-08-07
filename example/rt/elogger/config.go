package elogger

import "github.com/viant/toolbox"

// Config represents a logger config
type Config struct {
	Port     string
	LogTypes []toolbox.FileLoggerConfig
}
