package log

import (
	"io"
	"log"
	"os"
)

type Log struct {
	debug  		bool
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

func InitLog(logfile string, debug bool) Log {
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
	return Log{debug: debug, Debuglogger: *d, Infologger: *i, Warnlogger: *c, Errorlogger: *e}
}