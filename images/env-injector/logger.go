/*
Copyright 2026 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"net"
	"os"

	"github.com/deckhouse/deckhouse/pkg/log"
	"github.com/spf13/cast"
)

// Logger is the application logging interface backed by deckhouse/pkg/log.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type routingLogger struct {
	stdout *log.Logger
	stderr *log.Logger
	syslog *log.Logger
}

func (l *routingLogger) Debug(msg string, args ...any) {
	l.stdout.Debug(msg, args...)
}

func (l *routingLogger) Info(msg string, args ...any) {
	l.stdout.Info(msg, args...)
	if l.syslog != nil {
		l.syslog.Info(msg, args...)
	}
}

func (l *routingLogger) Warn(msg string, args ...any) {
	l.stderr.Warn(msg, args...)
	if l.syslog != nil {
		l.syslog.Warn(msg, args...)
	}
}

func (l *routingLogger) Error(msg string, args ...any) {
	l.stderr.Error(msg, args...)
	if l.syslog != nil {
		l.syslog.Error(msg, args...)
	}
}

func newLogger() Logger {
	level := log.LogLevelFromStr(os.Getenv("VAULT_LOG_LEVEL"))

	handlerType := log.TextHandlerType
	if cast.ToBool(os.Getenv("VAULT_JSON_LOG")) {
		handlerType = log.JSONHandlerType
	}

	opts := []log.Option{
		log.WithLevel(level.Level()),
		log.WithHandlerType(handlerType),
	}

	stdout := log.NewLogger(append(opts, log.WithOutput(os.Stdout))...).With("app", "env-injector")
	stderr := log.NewLogger(append(opts, log.WithOutput(os.Stderr))...).With("app", "env-injector")

	r := &routingLogger{
		stdout: stdout,
		stderr: stderr,
	}

	if logServerAddr := os.Getenv("VAULT_ENV_LOG_SERVER"); logServerAddr != "" {
		writer, err := net.Dial("udp", logServerAddr)
		// We silently ignore syslog connection errors for the lack of a better solution
		if err == nil {
			r.syslog = log.NewLogger(append(opts, log.WithOutput(writer))...).With("app", "env-injector")
		}
	}

	log.SetDefault(stdout)

	return r
}
