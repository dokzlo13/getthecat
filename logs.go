package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/x-cray/logrus-prefixed-formatter"
	"io"
	"io/ioutil"
	"os"
)

// WriterHook is a hook that writes logs of specified LogLevels to specified Writer
type WriterHook struct {
	Writer    io.Writer
	LogLevels []log.Level
}

// Fire will be called when some logging function is called with current hook
// It will format log entry to string and write it to appropriate writer
func (hook *WriterHook) Fire(entry *log.Entry) error {
	line, err := entry.String()
	if err != nil {
		return err
	}
	_, err = hook.Writer.Write([]byte(line))
	return err
}

// Levels define on which log levels this hook would trigger
func (hook *WriterHook) Levels() []log.Level {
	return hook.LogLevels
}

// setupLogs adds hooks to send logs to different destinations depending on level
func setupLogs(debuglvl int, logfile string) {
	log.SetOutput(ioutil.Discard) // Send all logs to nowhere by default
	textFormatter := &prefixed.TextFormatter{
		ForceColors: true,
		DisableColors: false,
		ForceFormatting: true,
		FullTimestamp: true,
	}

	var writer *os.File
	switch debuglvl {
	case 0:
		log.SetLevel(log.InfoLevel)

		log.SetFormatter(&log.JSONFormatter{PrettyPrint:true, DisableTimestamp:false})
		file, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Warning("Error creating\\opening log file \"%s\", using STDERR", logfile)
			writer = os.Stdout
		} else {
			log.Warning("Using log file \"%s\"", logfile)
			writer = file
		}

		log.AddHook(&WriterHook{ // Send info and debug logs to stdout
			Writer: writer,
			LogLevels: []log.Level{
				log.PanicLevel,
				log.FatalLevel,
				log.ErrorLevel,
				log.WarnLevel,
				log.InfoLevel,
			},
		})
	case 1:
		log.SetLevel(log.DebugLevel)
		log.SetFormatter(textFormatter)
		log.AddHook(&WriterHook{
			Writer: os.Stderr,
			LogLevels: []log.Level{
				log.PanicLevel,
				log.FatalLevel,
				log.ErrorLevel,
				log.WarnLevel,
				log.InfoLevel,
				log.DebugLevel,
			},
		})
	case 2 :
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(textFormatter)
		log.AddHook(&WriterHook{
			Writer: os.Stderr,
			LogLevels: []log.Level{
				log.PanicLevel,
				log.FatalLevel,
				log.ErrorLevel,
				log.WarnLevel,
				log.InfoLevel,
				log.DebugLevel,
				log.TraceLevel,
			},
		})
	case 3 :
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(textFormatter)
		log.AddHook(&WriterHook{
			Writer: os.Stderr,
			LogLevels: []log.Level{
				log.PanicLevel,
				log.FatalLevel,
				log.ErrorLevel,
				log.WarnLevel,
				log.InfoLevel,
				log.DebugLevel,
				log.TraceLevel,
			},
		})
	}


}