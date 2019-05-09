package logs

import (
	"fmt"
	"log"
)

//Logs interface
type Logs interface {
	Debug(v ...interface{})
	Info(v ...interface{})
	Warn(v ...interface{})
	Error(v ...interface{})
	Panic(v ...interface{})
}

//LogLevel log level debug, info ,warning...
type LogLevel uint8

const (
	//DebugLvl debug
	DebugLvl LogLevel = iota
	//InfoLvl info
	InfoLvl
	//WarnLvl warnngin
	WarnLvl
	//ErrorLvl error
	ErrorLvl
)

//SimpleLog 一个简单的log日志，封装了go 的log
type SimpleLog struct {
	Level LogLevel
}

//SLog 简单日志
var SLog = &SimpleLog{DebugLvl}

//SetLevel 设置log等级
func (l *SimpleLog) SetLevel(level LogLevel) {
	l.Level = level
}

//Debug debug print
func (l *SimpleLog) Debug(v ...interface{}) {
	if l.Level <= DebugLvl {
		val := make([]interface{}, 0, len(v)+1)
		val = append(val, "[Debug]")
		val = append(val, v...)
		log.Println(val...)
	}
}

//Info debug print
func (l *SimpleLog) Info(v ...interface{}) {
	if l.Level <= InfoLvl {
		val := make([]interface{}, 0, len(v)+1)
		val = append(val, "[Info]")
		val = append(val, v...)
		log.Println(val...)
	}
}

//Warn debug print
func (l *SimpleLog) Warn(v ...interface{}) {
	if l.Level <= WarnLvl {
		val := make([]interface{}, 0, len(v)+1)
		val = append(val, "[Warn]")
		val = append(val, v...)
		log.Println(val...)
	}
}

//Error debug print
func (l *SimpleLog) Error(v ...interface{}) {
	if l.Level <= ErrorLvl {
		val := make([]interface{}, 0, len(v)+1)
		val = append(val, "[Error]")
		val = append(val, v...)
		log.Println(val...)
	}
}

//Panic debug print
func (l *SimpleLog) Panic(v ...interface{}) {
	log.Panicln(v...)
}

//String return name:val
func String(name string, val string) string {
	return name + ":" + val
}

//Bool return name:val
func Bool(name string, val bool) string {
	return fmt.Sprintf("%s:%t", name, val)
}

//Int32 return name:val
func Int32(name string, val int32) string {
	return fmt.Sprintf("%s:%d", name, val)
}

//Int64 return name:val
func Int64(name string, val int64) string {
	return fmt.Sprintf("%s:%d", name, val)
}

//Float32 return name:val
func Float32(name string, val float32) string {
	return fmt.Sprintf("%s:%f", name, val)
}

//Float64 return name:val
func Float64(name string, val float64) string {
	return fmt.Sprintf("%s:%f", name, val)
}

//Error return err string or "nil"
func Error(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
