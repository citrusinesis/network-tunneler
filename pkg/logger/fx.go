package logger

import (
	"strings"

	"go.uber.org/fx/fxevent"
)

type FxLogger struct {
	logger Logger
}

func NewFxLogger(logger Logger) fxevent.Logger {
	return &FxLogger{
		logger: logger.With(String("component", "fx")),
	}
}

func (l *FxLogger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.OnStartExecuting:
		l.logger.Debug("OnStart hook executing",
			String("callee", e.FunctionName),
			String("caller", e.CallerName),
		)
	case *fxevent.OnStartExecuted:
		if e.Err != nil {
			l.logger.Error("OnStart hook failed",
				String("callee", e.FunctionName),
				String("caller", e.CallerName),
				Error(e.Err),
			)
		} else {
			l.logger.Debug("OnStart hook executed",
				String("callee", e.FunctionName),
				String("caller", e.CallerName),
				Duration("runtime", e.Runtime),
			)
		}
	case *fxevent.OnStopExecuting:
		l.logger.Debug("OnStop hook executing",
			String("callee", e.FunctionName),
			String("caller", e.CallerName),
		)
	case *fxevent.OnStopExecuted:
		if e.Err != nil {
			l.logger.Error("OnStop hook failed",
				String("callee", e.FunctionName),
				String("caller", e.CallerName),
				Error(e.Err),
			)
		} else {
			l.logger.Debug("OnStop hook executed",
				String("callee", e.FunctionName),
				String("caller", e.CallerName),
				Duration("runtime", e.Runtime),
			)
		}
	case *fxevent.Supplied:
		l.logger.Debug("supplied",
			String("type", e.TypeName),
			Any("module", e.ModuleName),
		)
	case *fxevent.Provided:
		for _, rtype := range e.OutputTypeNames {
			l.logger.Debug("provided",
				String("constructor", e.ConstructorName),
				Any("module", e.ModuleName),
				String("type", rtype),
			)
		}
	case *fxevent.Decorated:
		for _, rtype := range e.OutputTypeNames {
			l.logger.Debug("decorated",
				String("decorator", e.DecoratorName),
				Any("module", e.ModuleName),
				String("type", rtype),
			)
		}
	case *fxevent.Invoking:
		l.logger.Debug("invoking",
			String("function", e.FunctionName),
			Any("module", e.ModuleName),
		)
	case *fxevent.Invoked:
		if e.Err != nil {
			l.logger.Error("invoke failed",
				String("function", e.FunctionName),
				Any("module", e.ModuleName),
				Error(e.Err),
			)
		}
	case *fxevent.Stopping:
		l.logger.Info("received signal", String("signal", strings.ToUpper(e.Signal.String())))
	case *fxevent.Stopped:
		if e.Err != nil {
			l.logger.Error("stop failed", Error(e.Err))
		}
	case *fxevent.RollingBack:
		l.logger.Error("start failed, rolling back", Error(e.StartErr))
	case *fxevent.RolledBack:
		if e.Err != nil {
			l.logger.Error("rollback failed", Error(e.Err))
		}
	case *fxevent.Started:
		if e.Err != nil {
			l.logger.Error("start failed", Error(e.Err))
		} else {
			l.logger.Info("started")
		}
	case *fxevent.LoggerInitialized:
		if e.Err != nil {
			l.logger.Error("custom logger initialization failed", Error(e.Err))
		} else {
			l.logger.Debug("initialized custom fxevent.Logger", String("constructor", e.ConstructorName))
		}
	}
}
