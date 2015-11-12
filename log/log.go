package log

import (
	"io"
	"log"
	"log/syslog"
	"os"
)

var logger SlapLogger

type SlapLogger interface {
	Debug(string)
	Info(string)
	Warn(string)
	Error(string)
}

type SysLogger struct {
	debug        bool
	SyslogWriter *syslog.Writer
}

func (l *SysLogger) Debug(line string) {
	if l.debug == true {
		l.SyslogWriter.Debug("DEBUG: " + line)
	}
}

func (l *SysLogger) Info(line string) {
	l.SyslogWriter.Info("INFO : " + line)
}

func (l *SysLogger) Warn(line string) {
	l.SyslogWriter.Warning("WARN : " + line)
}

func (l *SysLogger) Error(line string) {
	l.SyslogWriter.Err("ERROR : " + line)
}

type Log struct {
	debug       bool
	Debuglogger log.Logger
	Infologger  log.Logger
	Warnlogger  log.Logger
	Errorlogger log.Logger
}

func (l *Log) Debug(line string) {
	if l.debug == true {
		l.Debuglogger.Println(line)
	}
}

func (l *Log) Info(line string) {
	l.Infologger.Println(line)
}

func (l *Log) Warn(line string) {
	l.Warnlogger.Println(line)
}

func (l *Log) Error(line string) {
	l.Errorlogger.Println(line)
}

func InitLog(log_syslog bool, logfile string, debug bool) {
	if log_syslog == true {
		slog, err := syslog.New(syslog.LOG_ALERT|syslog.LOG_LOCAL0, "slappy")
		if err != nil {
			panic(err)
		}
		logger = &SysLogger{debug: debug, SyslogWriter: slog}
	} else {
		var logwriter io.Writer = os.Stdout
		if logfile != "" {
			f, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				panic(err)
			}
			logwriter = io.MultiWriter(f, os.Stdout)
		}

		d := log.New(logwriter, "DEBUG: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC)
		i := log.New(logwriter, "INFO : ", log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC)
		c := log.New(logwriter, "WARN : ", log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC)
		e := log.New(logwriter, "ERROR: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC)
		logger = &Log{debug: debug, Debuglogger: *d, Infologger: *i, Warnlogger: *c, Errorlogger: *e}
	}
}

func Logger() SlapLogger {
	return logger
}
