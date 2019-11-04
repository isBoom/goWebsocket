package mylogger

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strings"
	"time"
)

type LogLever int

const (
	UNKNOW  LogLever = iota
	DEBUG   LogLever = iota
	TRACE   LogLever = iota
	INFO    LogLever = iota
	WARNING LogLever = iota
	ERROR   LogLever = iota
	FATAL   LogLever = iota
)

type Logger struct {
	Lever             LogLever `json:"lever"`
	UserInfo          *os.File `json:"userInfo"`
	SystemErr         *os.File `json:"systemErr"`
	MaxSize           int64    `json:"maxSize"`
	UserInfoFileName  string   `json:"userInfoFileName"`
	SystemErrFileName string   `json:"systemErrFileName"`
}

//格式化日志级别
func parseLogLeverToLogLever(s string) (LogLever, error) {
	s = strings.ToLower(s)
	switch s {
	case "debug":
		return DEBUG, nil
	case "trace":
		return TRACE, nil
	case "info":
		return INFO, nil
	case "warning":
		return WARNING, nil
	case "error":
		return ERROR, nil
	case "fatal":
		return FATAL, nil
	default:
		return UNKNOW, fmt.Errorf("无效的日志级别")
	}
}
func parseLogLeverToString(s LogLever) string {
	switch s {
	case TRACE:
		return "trace"
	case INFO:
		return "info"
	case WARNING:
		return "warning"
	case ERROR:
		return "error"
	case FATAL:
		return "fatal"
	default:
		return "debug"
	}
}

//返回日志对象
func NewLog(leverStr string) *Logger {
	var err error
	var lever LogLever
	lever, err = parseLogLeverToLogLever(leverStr)
	logObj := &Logger{
		Lever:             lever,
		UserInfoFileName:  "userInfo.log",
		SystemErrFileName: "systemErr.log",
		MaxSize:           (1 << 20) * 100,
	}
	//创建用户消息日志
	logObj.UserInfo, err = os.OpenFile(logObj.UserInfoFileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		panic(fmt.Sprintf("警告,打开用户消息日志文件失败!!!", err))
	}
	//创建系统错误日志
	logObj.SystemErr, err = os.OpenFile(logObj.SystemErrFileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		panic(fmt.Sprintf("警告,打开错误日志文件失败!!!", err))
	}
	//拦截ctrl+c 关闭文件流
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt)
		<-c
		logObj.UserInfo.Close()
		logObj.SystemErr.Close()
		os.Exit(0)
	}()

	if err != nil {
		panic(err)
	}
	return logObj
}
func (l Logger) enable(leverStr LogLever) bool {
	return l.Lever <= leverStr
}
func (l *Logger) log(lever LogLever, w io.Writer, msg string, a ...interface{}) {

	if l.enable(lever) {
		var err error
		now := time.Now()
		msg = fmt.Sprintf(msg, a...)
		funcName, fileName, line := GetInfo(3)
		if lever <= TRACE { //控制台级别
			fmt.Fprintf(w, "[%s] [%s] [%s:%s:%d] ===>%s \n", now.Format("2006-01-02 15:04:05"), parseLogLeverToString(lever), fileName, funcName, line, msg)
		} else if lever <= INFO { //用户信息级别
			if stat, err := l.UserInfo.Stat(); err != nil {
				funcName, fileName, line := GetInfo(1)
				fmt.Printf("[%s:%s:%d:%v]\n", funcName, fileName, line, err)
			} else if stat.Size() > l.MaxSize {
				l.UserInfo.Close()
				os.Rename(l.UserInfoFileName, fmt.Sprintf("%s[%v].bak", l.UserInfoFileName, now.Format("20060102150405")))
				l.UserInfo, err = os.OpenFile(l.UserInfoFileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
				if err != nil {
					funcName, fileName, line := GetInfo(1)
					fmt.Printf("[%s:%s:%d:用户信息日志未成功新建%v]\n", funcName, fileName, line, err)
				}
			}
			if _, err = fmt.Fprintf(l.UserInfo, "[%s] [%s] [%s:%s:%d] ===>%s \n", now.Format("2006-01-02 15:04:05"), parseLogLeverToString(lever), fileName, funcName, line, msg); err != nil {
				funcName, fileName, line := GetInfo(1)
				fmt.Printf("[%s:%s:%d:用户信息日志写入失败%v]\n", funcName, fileName, line, err)
			}

		} else { //err级别
			if stat, err := l.SystemErr.Stat(); err != nil {
				fmt.Println(err)
			} else if stat.Size() > l.MaxSize {
				l.SystemErr.Close()
				os.Rename(l.SystemErrFileName, fmt.Sprintf("%s[%v].bak", l.SystemErrFileName, now.Format("20060102150405")))
				l.SystemErr, err = os.OpenFile(l.SystemErrFileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
				if err != nil {
					funcName, fileName, line := GetInfo(1)
					fmt.Printf("[%s:%s:%d:用户信息日志未成功新建%v]\n", funcName, fileName, line, err)
				}
			}
			if _, err = fmt.Fprintf(l.SystemErr, "[%s] [%s] [%s:%s:%d] ===>%s \n", now.Format("2006-01-02 15:04:05"), parseLogLeverToString(lever), fileName, funcName, line, msg); err != nil {
				funcName, fileName, line := GetInfo(1)
				fmt.Printf("[%s:%s:%d:用户信息日志写入失败%v]\n", funcName, fileName, line, err)
			}
		}
	}

}
func (l *Logger) Debug(msg string, a ...interface{}) {
	l.log(DEBUG, os.Stdout, msg, a...)
}
func (l *Logger) Trace(msg string, a ...interface{}) {
	l.log(TRACE, l.UserInfo, msg, a...)
}
func (l *Logger) Info(msg string, a ...interface{}) {
	l.log(INFO, l.UserInfo, msg, a...)
}
func (l *Logger) Warning(msg string, a ...interface{}) {
	l.log(WARNING, l.UserInfo, msg, a...)
	l.log(WARNING, l.SystemErr, msg, a...)
}
func (l *Logger) Error(msg string, a ...interface{}) {
	l.log(ERROR, l.SystemErr, msg, a...)
}
func (l *Logger) Fatal(msg string, a ...interface{}) {
	l.log(FATAL, l.SystemErr, msg, a...)
	os.Exit(0)
}
func GetInfo(skip int) (string, string, int) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		fmt.Println("runtime.CallerErr", ok)
		return "", "", 0
	} else {
		funcName := runtime.FuncForPC(pc).Name()
		funcName = strings.Split(funcName, ".")[1]
		fileName := path.Base(file)
		return funcName, fileName, line
	}
}
