package log

import (
	"fmt"
	"time"
)

type LogConfig struct {
	Type              string // stderr/std/file
	Level             string // DEBUG/INFO/WARNING/ERROR/FATAL
	FileName          string
	Prefix            string
	FileRotateCount   int
	FileRotateSize    uint64
	FileFlushDuration time.Duration
}

func initFromConfig(log *Logger, fb *FileBackend, config LogConfig) error {
	log.prefix = config.Prefix
	if config.Type == "stderr" || config.Type == "std" {
		log.LogToStderr()
		log.SetSeverity(config.Level)
		return nil
	}

	var err error
	if config.Type == "file" {
		if fb, err = NewFileBackend(config.FileName); err != nil {
			return err
		}

		log.SetLogging(config.Level, fb)
		fb.Rotate(config.FileRotateCount, config.FileRotateSize)
		fb.SetFlushDuration(config.FileFlushDuration)
	} else {
		return fmt.Errorf("unknown log type: %s", config.Type)
	}

	return nil
}

func Init(config LogConfig) error {
	return initFromConfig(&logging, fileback, config)
}

func NewLoggerFromConfig(config LogConfig) (Logger, error) {
	var log Logger
	var fb *FileBackend = nil
	err := initFromConfig(&log, fb, config)
	return log, err
}
