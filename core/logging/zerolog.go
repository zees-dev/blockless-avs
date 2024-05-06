package logging

import (
	"fmt"
	"os"
	"time"

	"github.com/Layr-Labs/eigensdk-go/logging"
	"github.com/rs/zerolog"
)

type LogLevel string

const (
	Development LogLevel = "development" // prints debug and above
	Production  LogLevel = "production"  // prints info and above
)

type ZeroLogger struct {
	logger *zerolog.Logger
}

var _ logging.Logger = (*ZeroLogger)(nil)

func NewZeroLogger(env LogLevel) *ZeroLogger {
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	if env == Production {
		logger := zerolog.New(output).With().Timestamp().Logger().Level(zerolog.InfoLevel)
		return &ZeroLogger{logger: &logger}
	} else if env == Development {
		logger := zerolog.New(output).With().Timestamp().Logger().Level(zerolog.DebugLevel)
		return &ZeroLogger{logger: &logger}
	} else {
		panic(fmt.Sprintf("Unknown environment. Expected %s or %s. Received %s.", Development, Production, env))
	}
}

// Inner gets the inner logger.
func (z *ZeroLogger) Inner() *zerolog.Logger {
	return z.logger
}

func (z *ZeroLogger) Debug(msg string, tags ...any) {
	z.logger.Debug().Msgf(msg, tags...)
}

func (z *ZeroLogger) Info(msg string, tags ...any) {
	z.logger.Info().Msgf(msg, tags...)
}

func (z *ZeroLogger) Warn(msg string, tags ...any) {
	z.logger.Warn().Msgf(msg, tags...)
}

func (z *ZeroLogger) Error(msg string, tags ...any) {
	z.logger.Error().Msgf(msg, tags...)
}

func (z *ZeroLogger) Fatal(msg string, tags ...any) {
	z.logger.Fatal().Msgf(msg, tags...)
}

func (z *ZeroLogger) Debugf(template string, args ...interface{}) {
	z.logger.Debug().Msgf(template, args...)
}

func (z *ZeroLogger) Infof(template string, args ...interface{}) {
	z.logger.Info().Msgf(template, args...)
}

func (z *ZeroLogger) Warnf(template string, args ...interface{}) {
	z.logger.Warn().Msgf(template, args...)
}

func (z *ZeroLogger) Errorf(template string, args ...interface{}) {
	z.logger.Error().Msgf(template, args...)
}

func (z *ZeroLogger) Fatalf(template string, args ...interface{}) {
	z.logger.Fatal().Msgf(template, args...)
}

// With does not apply to zerolog logger
func (z *ZeroLogger) With(tags ...any) logging.Logger {
	return &ZeroLogger{
		// logger: z.logger.Sugar().With(tags...).Desugar(),
		logger: z.logger,
	}
}
