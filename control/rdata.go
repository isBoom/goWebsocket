package control

import (
	"encoding/json"
	"errors"
	"net/http"
)

type SysteMsg struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
}

func SendMsg(w http.ResponseWriter, status int, msg string) error {
	str := &SysteMsg{
		Status: status,
		Msg:    msg,
	}
	temp, err := json.Marshal(str)
	if err != nil {
		err = errors.New("json化失败")
		return err
	}
	w.Write(temp)
	return nil
}

/*
0 其他错误
10 cookie错误 提示alert
20 账号已登录 提示alert
30 没有cookie 使其跳转登录并且不提示alert
100 普通的来自服务器的消息
110 在线用户信息
120 谁上线了
200 普通用户消息
210 发送图片
310 更改头像
311 修改头像失败
312 修改图像成功(群)
*/
