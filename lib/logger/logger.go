package logger

import (
	"os"
	"time"

	"github.com/evalphobia/logrus_sentry"
	"github.com/sirupsen/logrus"
)

// Contract ...
type Contract interface {
	Sentry() *logrus.Logger
	File() *logrus.Logger
	FromDefault() *logrus.Logger
}

// logs ...
type logs struct {
	Logrus      *logrus.Logger
	DefaultType string
	Source      string
}

// New instantiate the logger package
func New(tp, source string) Contract {
	return &logs{
		Logrus:      logrus.New(),
		DefaultType: tp,
		Source:      source,
	}
}

func (th logs) FromDefault() *logrus.Logger {
	var log *logrus.Logger
	switch def := th.DefaultType; def {
	case "file":
		log = th.File()
	case "sentry":
		log = th.Sentry()
	}

	// log.SetFormatter(&logrus.JSONFormatter{})
	return log
}

// File is a function to set logrus with file
func (th logs) File() *logrus.Logger {
	var (
		err  error
		file *os.File
	)
	path := th.Source
	// if file exist then open and append/write into log file.
	if _, err = os.Stat(path); err == nil {
		file, err = os.OpenFile(path, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if err == nil {
			th.Logrus.Out = file

			return th.Logrus
		}
	}

	file, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0755)
	if err == nil {
		th.Logrus.Out = file
	} else {
		th.Logrus.Info("Failed to log to file, using default stderr")
	}

	defer file.Close()

	return th.Logrus
}

// Sentry is a function for setting Sentry.io logger
func (th logs) Sentry() *logrus.Logger {
	var (
		err  error
		hook *logrus_sentry.SentryHook
	)
	hook, err = logrus_sentry.NewAsyncSentryHook(th.Source, []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.DebugLevel,
		logrus.InfoLevel,
		logrus.WarnLevel,
	})

	hook.Timeout = 10 * time.Second

	if err == nil {
		th.Logrus.Hooks.Add(hook)
	}

	return th.Logrus
}
