package logger

import (
	"errors"
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Format string

const (
	JSON    Format = "json"
	CONSOLE Format = "console"
)

var allFormats = []Format{JSON, TEXT}

func MapFormat(input string) (Format, error) {
	var format = Format(input)
	switch format {
	case JSON, CONSOLE:
		return format, nil
	case "text":
		return CONSOLE, nil
	default:
		return format, fmt.Errorf("given log format: %s, doesn't match with any of %v", format, allFormats)
	}
}

func (f Format) ToZapEncoder() (zapcore.Encoder, error) {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.MessageKey = "message"
	switch f {
	case JSON:
		return zapcore.NewJSONEncoder(encoderConfig), nil
	case CONSOLE:
		return zapcore.NewConsoleEncoder(encoderConfig), nil
	default:
		return nil, errors.New("unknown encoder")
	}
}
