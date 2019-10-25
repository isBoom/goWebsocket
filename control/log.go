package control

import (
	"fmt"
	"os"
	"os/signal"
	"time"
)

var (
	File *os.File
	now  = time.Now()
)

func init() {

	//拦截ctrl+c
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	//关闭文件流
	go func() {
		<-c
		File.Close()
		os.Exit(0)
	}()
	currentTime := fmt.Sprintf("%d-%d-%d", now.Year(), now.Month(), now.Day())
	//日志文件是否存在
	if _, err := os.Stat(fmt.Sprintf("%s.log", currentTime)); err != nil {
		if os.IsNotExist(err) {
			var errFile error
			//创建日志文件
			File, errFile = os.Create(fmt.Sprintf("%s.log", currentTime))
			if errFile != nil {
				fmt.Println("警告,创建日志文件失败!!!")
			}
		} else {
			fmt.Println("警告,日志开启失!!!")
		}
	} else {
		//追加日志文件内容
		var errFile error
		File, errFile = os.OpenFile(fmt.Sprintf("%s.log", currentTime), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0660)
		if errFile != nil {
			fmt.Println("警告,打开日志文件失败!!!")
			return
		}
	}
}

//记录日志
func Log(msg string) {
	data := fmt.Sprintf("%d-%d-%d-%d:%d:%d-->%s\n", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), msg)
	_, err := File.Write([]byte(data))
	if err != nil {
		fmt.Println(err)
	}
}
