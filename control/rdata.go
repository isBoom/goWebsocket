package control

import (
	"encoding/json"
	"net/http"
	"websocket/model"
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
		model.Log.Warning("json.Marshal %v", err)
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
233 离线消息
310 更改头像
311 修改头像失败
312 修改图像成功(群)
400 普通私信  401是给消息发送者的
410 私信图片  411是给消息发送者的
500 添加好友
510 查无此人
520 有人添加你为好友
530 查看自己的好友请求
540 所有在线好友列表
550 拒绝
560 添加成功
570 添加失败
580 已经是好友重复添加
*/
