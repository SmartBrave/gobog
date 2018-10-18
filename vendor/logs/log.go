// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Usage:
//
// import "github.com/astaxie/beego/logs"
//
//	log := NewXLogger(10000)
//	log.SetXLogger("console", "")
//
//	> the first params stand for how many channel
//
// Use it like this:
//
//	log.Trace("trace")
//	log.Info("info")
//	log.Warn("warning")
//	log.Debug("debug")
//	log.Critical("critical")
//
//  more docs http://beego.me/docs/module/logs.md
package logs

import (
	"fmt"
	"path"
	"runtime"
	"sync"
)

// RFC5424 log message levels.
const (
	LevelEmergency = iota
	LevelAlert
	LevelCritical
	LevelError
	LevelWarning
	LevelNotice
	LevelInformational
	LevelDebug
)

// Legacy loglevel constants to ensure backwards compatibility.
//
// Deprecated: will be removed in 1.5.0.
const (
	LevelInfo  = LevelInformational
	LevelTrace = LevelDebug
	LevelWarn  = LevelWarning
)

type loggerType func() XLoggerInterface

// XLoggerInterface defines the behavior of a log provider.
type XLoggerInterface interface {
	Init(config string) error
	WriteMsg(msg string, level int) error
	Destroy()
	Flush()
}

var adapters = make(map[string]loggerType)

// Register makes a log provide available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, log loggerType) {
	if log == nil {
		panic("logs: Register provide is nil")
	}
	if _, dup := adapters[name]; dup {
		panic("logs: Register called twice for provider " + name)
	}
	adapters[name] = log
}

// XLogger is default logger in beego application.
// it can contain several providers and log message into all providers.
type XLogger struct {
	lock                sync.Mutex
	level               int
	enableFuncCallDepth bool
	loggerFuncCallDepth int
	asynchronous        bool
	msg                 chan *logMsg
	outputs             map[string]XLoggerInterface
	extra				string
}

type logMsg struct {
	level int
	msg   string
}

// NewXLogger returns a new XLogger.
// channellen means the number of messages in chan.
// if the buffering chan is full, logger adapters write to file or other way.
func NewXLogger(channellen int64) *XLogger {
	bl := new(XLogger)
	bl.level = LevelDebug
	bl.loggerFuncCallDepth = 2
	bl.msg = make(chan *logMsg, channellen)
	bl.outputs = make(map[string]XLoggerInterface)
	return bl
}

func (bl *XLogger) Async() *XLogger {
	bl.asynchronous = true
	go bl.startXLogger()
	return bl
}

// SetXLogger provides a given logger adapter into XLogger with config string.
// config need to be correct JSON as string: {"interval":360}.
func (bl *XLogger) SetXLogger(adaptername string, config string) error {
	bl.lock.Lock()
	defer bl.lock.Unlock()
	if log, ok := adapters[adaptername]; ok {
		lg := log()
		err := lg.Init(config)
		bl.outputs[adaptername] = lg
		if err != nil {
			fmt.Println("logs.XLogger.SetXLogger: " + err.Error())
			return err
		}
	} else {
		return fmt.Errorf("logs: unknown adaptername %q (forgotten Register?)", adaptername)
	}
	return nil
}

// remove a logger adapter in XLogger.
func (bl *XLogger) DelXLogger(adaptername string) error {
	bl.lock.Lock()
	defer bl.lock.Unlock()
	if lg, ok := bl.outputs[adaptername]; ok {
		lg.Destroy()
		delete(bl.outputs, adaptername)
		return nil
	} else {
		return fmt.Errorf("logs: unknown adaptername %q (forgotten Register?)", adaptername)
	}
}

func (bl *XLogger) writerMsg(loglevel int, msg string) error {
	lm := new(logMsg)
	lm.level = loglevel
	if bl.enableFuncCallDepth {
		_, file, line, ok := runtime.Caller(bl.loggerFuncCallDepth)
		if !ok {
			file = "???"
			line = 0
		}
		_, filename := path.Split(file)
		lm.msg = fmt.Sprintf("[%s:%s:%d] %s",bl.extra, filename, line, msg)
	} else {
		lm.msg = fmt.Sprintf("[%s] %s",bl.extra,msg)
	}
	if bl.asynchronous {
		bl.msg <- lm
	} else {
		for name, l := range bl.outputs {
				//fmt.Println("111111111111", lm.msg)

			err := l.WriteMsg(lm.msg, lm.level)
			if err != nil {
				fmt.Println("unable to WriteMsg to adapter:", name, err)
				return err
			}
		}
	}
	return nil
}

// Set log message level.
//
// If message level (such as LevelDebug) is higher than logger level (such as LevelWarning),
// log providers will not even be sent the message.
func (bl *XLogger) SetLevel(l int) {
	bl.level = l
}
// set log extra
func (bl *XLogger) SetExtra(extra string) {
	bl.extra = extra
}
// set log funcCallDepth
func (bl *XLogger) SetLogFuncCallDepth(d int) {
	bl.loggerFuncCallDepth = d
}

// get log funcCallDepth for wrapper
func (bl *XLogger) GetLogFuncCallDepth() int {
	return bl.loggerFuncCallDepth
}

// enable log funcCallDepth
func (bl *XLogger) EnableFuncCallDepth(b bool) {
	bl.enableFuncCallDepth = b
}

// start logger chan reading.
// when chan is not empty, write logs.
func (bl *XLogger) startXLogger() {
	for {
		select {
		case bm := <-bl.msg:
			for _, l := range bl.outputs {
				err := l.WriteMsg(bm.msg, bm.level)
				if err != nil {
					fmt.Println("ERROR, unable to WriteMsg:", err)
				}
			}
		}
	}
}

// Log EMERGENCY level message.
func (bl *XLogger) Emergency(format string, v ...interface{}) {
	if LevelEmergency > bl.level {
		return
	}
	msg := fmt.Sprintf("[M] "+format, v...)
	bl.writerMsg(LevelEmergency, msg)
}

// Log ALERT level message.
func (bl *XLogger) Alert(format string, v ...interface{}) {
	if LevelAlert > bl.level {
		return
	}
	msg := fmt.Sprintf("[A] "+format, v...)
	bl.writerMsg(LevelAlert, msg)
}

// Log CRITICAL level message.
func (bl *XLogger) Critical(format string, v ...interface{}) {
	if LevelCritical > bl.level {
		return
	}
	msg := fmt.Sprintf("[C] "+format, v...)
	bl.writerMsg(LevelCritical, msg)
}

// Log ERROR level message.
func (bl *XLogger) Error(format string, v ...interface{}) {
	if LevelError > bl.level {
		return
	}
	msg := fmt.Sprintf("[E] "+format, v...)
	bl.writerMsg(LevelError, msg)
}

// Log WARNING level message.
func (bl *XLogger) Warning(format string, v ...interface{}) {
	if LevelWarning > bl.level {
		return
	}
	msg := fmt.Sprintf("[W] "+format, v...)
	bl.writerMsg(LevelWarning, msg)
}

// Log NOTICE level message.
func (bl *XLogger) Notice(format string, v ...interface{}) {
	if LevelNotice > bl.level {
		return
	}
	msg := fmt.Sprintf("[N] "+format, v...)
	bl.writerMsg(LevelNotice, msg)
}

// Log INFORMATIONAL level message.
func (bl *XLogger) Informational(format string, v ...interface{}) {
	if LevelInformational > bl.level {
		return
	}
	msg := fmt.Sprintf("[I] "+format, v...)
	bl.writerMsg(LevelInformational, msg)
}

// Log DEBUG level message.
func (bl *XLogger) Debug(format string, v ...interface{}) {
	if LevelDebug > bl.level {
		return
	}
	msg := fmt.Sprintf("[D] "+format, v...)
	bl.writerMsg(LevelDebug, msg)
}

// Log WARN level message.
// compatibility alias for Warning()
func (bl *XLogger) Warn(format string, v ...interface{}) {
	if LevelWarning > bl.level {
		return
	}
	msg := fmt.Sprintf("[W] "+format, v...)
	bl.writerMsg(LevelWarning, msg)
}

// Log INFO level message.
// compatibility alias for Informational()
func (bl *XLogger) Info(format string, v ...interface{}) {
	if LevelInformational > bl.level {
		return
	}
	msg := fmt.Sprintf("[I] "+format, v...)
	bl.writerMsg(LevelInformational, msg)
}

// Log TRACE level message.
// compatibility alias for Debug()
func (bl *XLogger) Trace(format string, v ...interface{}) {
	if LevelDebug > bl.level {
		return
	}
	msg := fmt.Sprintf("[D] "+format, v...)
	bl.writerMsg(LevelDebug, msg)
}

// flush all chan data.
func (bl *XLogger) Flush() {
	for _, l := range bl.outputs {
		l.Flush()
	}
}

// close logger, flush all chan data and destroy all adapters in XLogger.
func (bl *XLogger) Close() {
	for {
		if len(bl.msg) > 0 {
			bm := <-bl.msg
			for _, l := range bl.outputs {
				err := l.WriteMsg(bm.msg, bm.level)
				if err != nil {
					fmt.Println("ERROR, unable to WriteMsg (while closing logger):", err)
				}
			}
			continue
		}
		break
	}
	for _, l := range bl.outputs {
		l.Flush()
		l.Destroy()
	}
}