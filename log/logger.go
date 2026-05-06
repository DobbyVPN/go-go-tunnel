package log

import (
	"fmt"
	"log/slog"
)

var lg TrustTunnelLogger

type TrustTunnelLogger struct {
	logger *slog.Logger
}

func InitLogger(logger *slog.Logger) {
	lg = TrustTunnelLogger{
		logger: logger,
	}
}

func Debugf(format string, args ...any) {
	lg.logger.Debug(fmt.Sprintf(format, args...))
}

func Infof(format string, args ...any) {
	lg.logger.Info(fmt.Sprintf(format, args...))
}

func Warnf(format string, args ...any) {
	lg.logger.Warn(fmt.Sprintf(format, args...))
}

func Errorf(format string, args ...any) {
	lg.logger.Error(fmt.Sprintf(format, args...))
}
