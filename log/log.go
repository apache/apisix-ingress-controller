package log

import (
	"bufio"
	"fmt"
	"github.com/api7/ingress-controller/pkg/config"
	"github.com/sirupsen/logrus"
	"log/syslog"
	"os"
	"runtime"
	"strings"
)

var logEntry *logrus.Entry

func GetLogger() *logrus.Entry {
	if logEntry == nil {
		var log = logrus.New()
		setNull(log)
		log.SetLevel(logrus.DebugLevel)
		if config.ENV != config.LOCAL {
			log.SetLevel(logrus.InfoLevel)
		}
		log.SetFormatter(&logrus.JSONFormatter{})
		logEntry = log.WithFields(logrus.Fields{
			"app": "ingress-controller",
		})
		hook, err := createHook("udp", fmt.Sprintf("%s:514", config.SyslogServer),
			syslog.LOG_LOCAL4, "ingress-controller")
		if err != nil {
			panic("failed to create log hook " + config.SyslogServer)
		}
		log.AddHook(hook)
	}
	return logEntry
}

func setNull(log *logrus.Logger) {
	src, err := os.OpenFile(os.DevNull, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		fmt.Println("err", err)
	}
	writer := bufio.NewWriter(src)
	log.SetOutput(writer)
}

type SysLogHook struct {
	Writer    *syslog.Writer
	NetWork   string
	Raddr     string
	Formatter func(file, function string, line int) string
	LineName  string
}

func createHook(network, raddr string, priority syslog.Priority, tag string) (*SysLogHook, error) {
	if w, err := syslog.Dial(network, raddr, priority, tag); err != nil {
		return nil, err
	} else {
		return &SysLogHook{w, network, raddr,
			func(file, function string, line int) string {
				return fmt.Sprintf("%s:%d", file, line)
			},
			"line",
		}, nil
	}
}

func (hook *SysLogHook) Fire(entry *logrus.Entry) error {
	//entry.Data[hook.LineName] = hook.Formatter(findCaller(5))
	en := entry.WithField(hook.LineName, hook.Formatter(findCaller(5)))
	en.Level = entry.Level
	en.Message = entry.Message
	en.Time = entry.Time
	line, err := en.String()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read entry, %v", err)
		return err
	}

	switch en.Level {
	case logrus.PanicLevel:
		hook.Writer.Crit(line)
		localPrint(line)
		return nil
	case logrus.FatalLevel:
		hook.Writer.Crit(line)
		localPrint(line)
		return nil
	case logrus.ErrorLevel:
		hook.Writer.Err(line)
		localPrint(line)
		return nil
	case logrus.WarnLevel:
		hook.Writer.Warning(line)
		localPrint(line)
		return nil
	case logrus.InfoLevel:
		hook.Writer.Info(line)
		localPrint(line)
		return nil
	case logrus.DebugLevel:
		hook.Writer.Debug(line)
		localPrint(line)
		return nil
	default:
		return nil
	}
}

func localPrint(line string) {
	if config.ENV != config.BETA && config.ENV != config.PROD && config.ENV != config.HBPROD {
		fmt.Print(line)
	}
}

func (hook *SysLogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func findCaller(skip int) (string, string, int) {
	var (
		pc       uintptr
		file     string
		function string
		line     int
	)
	for i := 0; i < 10; i++ {
		pc, file, line = getCaller(skip + i)
		if !strings.HasPrefix(file, "logrus") {
			break
		}
	}
	if pc != 0 {
		frames := runtime.CallersFrames([]uintptr{pc})
		frame, _ := frames.Next()
		function = frame.Function
	}
	return file, function, line
}

func getCaller(skip int) (uintptr, string, int) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return 0, "", 0
	}
	n := 0
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			n += 1
			if n >= 2 {
				file = file[i+1:]
				break
			}
		}
	}
	return pc, file, line
}
