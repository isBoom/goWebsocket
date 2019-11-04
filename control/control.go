package control

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"model"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	xx = "fsjiamkfasifjaiodmasdkaso"
)

func Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, POST")
	r.ParseForm()
	loginInfo, err := model.LoginInfo(r.PostForm.Get("userName"), r.PostForm.Get("userPassword"))
	if err != nil {
		if e := SendMsg(w, 0, fmt.Sprint(err)); e != nil {
			model.Log.Warning("某用户以账号[%d][%s]密码[%s]登录但是发生了错误且消息没有传达  %v", loginInfo.UserId, r.PostForm.Get("userName"), r.PostForm.Get("userPassword"), e)
		} else {
			model.Log.Info("某用户以账号[%d][%s]密码[%s]登录但是发生了错误  %v", loginInfo.UserId, r.PostForm.Get("userName"), r.PostForm.Get("userPassword"), err)
		}
		return
	} else {
		//该用户是否在线
		uid, errAtoi := strconv.Atoi(strconv.FormatInt(loginInfo.UserId, 10))
		if errAtoi != nil {
			model.Log.Warning("strconv.Atoi%v", errAtoi)
		}
		if ClientMap[uid] != nil {
			SendMsg(w, 20, "该账户已登陆")
			model.Log.Info("账号[%d][%s]已登录,有人挤他", uid, r.PostForm.Get("userName"))
			return
		} else {
			//不在线则允许登录并设置cookie
			c1 := http.Cookie{
				Name:   "userId",
				Value:  fmt.Sprint(loginInfo.UserId),
				Domain: "",
				Path:   "/",
				MaxAge: 86400 * 3,
			}

			c2 := http.Cookie{
				Name:   "verification",
				Value:  fmt.Sprintf("%x%x", md5.Sum([]byte(loginInfo.UserEmail)), md5.Sum([]byte(loginInfo.UserPassword))),
				Domain: "",
				Path:   "/",
				MaxAge: 86400 * 3,
			}
			w.Header().Add("Set-cookie", c1.String())
			w.Header().Add("Set-cookie", c2.String())
			//发送成功信息
			if e := SendMsg(w, 100, "登陆成功"); e != nil {
				model.Log.Warning("(%d)%s登陆成功但是消息没有传达到  %v", uid, r.PostForm.Get("userName"), e)
			} else {
				model.Log.Info("(%d)%s登陆成功", uid, r.PostForm.Get("userName"))
			}
		}
	}

}
func Registe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, POST")
	r.ParseForm()
	registeId, err := model.RegisteUser(r.PostForm.Get("userName"), r.PostForm.Get("userPassword"), r.PostForm.Get("userEmail"))
	if err != nil {
		if e := SendMsg(w, 0, fmt.Sprint(err)); e != nil {
			model.Log.Warning("有人注册但是发生了错误且没有传达 %v", e)
		} else {
			model.Log.Info("有人注册但是发生了错误 %v", err)
		}
	} else {
		c1 := http.Cookie{
			Name:   "userId",
			Value:  fmt.Sprint(registeId),
			Domain: "",
			Path:   "/",
			MaxAge: 86400 * 3,
		}
		c2 := http.Cookie{
			Name:   "verification",
			Value:  fmt.Sprintf("%x%x", md5.Sum([]byte(r.PostForm.Get("userEmail"))), md5.Sum([]byte(r.PostForm.Get("userPassword")))),
			Domain: "",
			Path:   "/",
			MaxAge: 86400 * 3,
		}
		w.Header().Add("Set-cookie", c1.String())
		w.Header().Add("Set-cookie", c2.String())
		if e := SendMsg(w, 100, "注册成功"); e != nil {
			model.Log.Warning("[%d][%s]注册账号成功但是没有发送给他这个消息 %v", registeId, r.PostForm.Get("userName"), e)
		} else {
			model.Log.Info("[%d][%s]注册了账号", registeId, r.PostForm.Get("userName"))
		}
	}
}
func ChangeUserHeadPortraitBox(c *Client, msgFromUser *MsgFromUser) {
	uid := c.UserInfo.Uid
	now := time.Now()
	nowTime := now.Format("2006-01-02_15:04:05")
	filename := fmt.Sprintf("Uid%d_%s.%s", uid, nowTime, msgFromUser.UserName[6:])
	index := strings.Index(msgFromUser.Msg, ",")
	base64Data, errDecodeString := base64.StdEncoding.DecodeString(msgFromUser.Msg[index+1:])
	if errDecodeString != nil {
		model.Log.Warning("errDecodeString %v", errDecodeString)
	}
	file, err := os.Create("/www/html/img/userHeadPortrait/" + filename)
	if err != nil {
		model.Log.Warning("os.Create %v", err)
	}
	defer file.Close()
	file.Write(base64Data)
	if errChange := model.ChangeUserHeadPortrait(uid, "https://xxxholic.top/img/userHeadPortrait/"+filename); err != nil {
		model.Log.Warning("model.ChangeUserHeadPortrait %v", errChange)
		temp, _ := json.Marshal(MsgFromUser{Status: 311, Msg: fmt.Sprint(errChange)})
		if err != nil {
			model.Log.Warning("json.Marshal %v", err)
		}
		c.Socket.WriteMessage(websocket.TextMessage, temp)
		return
	} else {
		//广播修改头像
		var msgToUserOnlie = &MsgToUserOnlie{
			Status: 312,
			Msg:    "修改了头像",
		}
		c.UserInfo.UserHeadPortrait = msgFromUser.Msg
		msgToUserOnlie.User = make([]UserSimpleData, 1)
		msgToUserOnlie.User[0] = UserSimpleData{
			Uid:              c.UserInfo.Uid,
			UserHeadPortrait: msgFromUser.Msg,
		}
		temp, err := json.Marshal(msgToUserOnlie)
		if err != nil {
			model.Log.Warning("json.Marshal %v", err)
		}
		Message <- temp
		model.Log.Info("[%d][%s]修改了头像", c.UserInfo.Uid, c.UserInfo.UserName)
	}
}
func PrivateChat(c *Client, msgFromUser *MsgFromUser) {
	fromId := c.UserInfo.Uid
	toId := msgFromUser.Uid
	if ClientMap[toId] == nil {
		//不在线 日后再写
		temp, _ := json.Marshal(MsgFromUser{Status: 0, Msg: "对方不在线"})
		c.Socket.WriteMessage(websocket.TextMessage, temp)
		return
	} else {
		if _, err := model.SelectUserId(strconv.Itoa(toId)); err != nil { //cookie不正确
			model.Log.Warning("model.SelectUserId %v", err)
			temp, _ := json.Marshal(MsgFromUser{Status: 0, Msg: "可能没这个人"})
			c.Socket.WriteMessage(websocket.TextMessage, temp)
			return
		} else {
			msg := msgFromUser.Msg
			//接收者
			temp, err := json.Marshal(MsgFromUser{Status: msgFromUser.Status, Uid: fromId, Msg: msg})
			if err != nil {
				model.Log.Warning("model.SelectUserId %v", err)
				return
			}
			if err := ClientMap[toId].Socket.WriteMessage(websocket.TextMessage, temp); err != nil {
				model.Log.Warning("ClientMap[toId].Socket.WriteMessageErr %v", err)
				return
			}
			//发送者
			temp, _ = json.Marshal(MsgFromUser{Status: (msgFromUser.Status + 1), Uid: toId, Msg: msg})
			if err := c.Socket.WriteMessage(websocket.TextMessage, temp); err != nil {
				model.Log.Warning("c.Socket.WriteMessageErr", err)
				return
			}
			model.Log.Info("[%d][%s]对[%d][%s]说[%s]", fromId, c.UserInfo.UserName, toId, ClientMap[toId].UserInfo.UserName, msg)
		}
	}
}
