package control

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
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
		//密码错误之类的
		log.Println(err)
		if e := SendMsg(w, 0, fmt.Sprint(err)); e != nil {
			Log(fmt.Sprintf("某用户以账号_(%d)%s_密码_%s_登录但是发生了错误且消息没有传达  %v", loginInfo.UserId, r.PostForm.Get("userName"), r.PostForm.Get("userPassword"), e))
		}
		Log(fmt.Sprintf("某用户以账号_(%d)%s_密码_%s_登录但是发生了错误  %v", loginInfo.UserId, r.PostForm.Get("userName"), r.PostForm.Get("userPassword"), err))
		return
	} else {
		//该用户是否在线
		uid, errAtoi := strconv.Atoi(strconv.FormatInt(loginInfo.UserId, 10))
		if errAtoi != nil {
			fmt.Println(errAtoi)
		}
		if ClientMap[uid] != nil {
			SendMsg(w, 20, "该账户已登陆")
			Log(fmt.Sprintf("账号_(%d)%s_已登录,有人挤他", uid, r.PostForm.Get("userName")))
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
				Log(fmt.Sprintf("(%d)%s登陆成功但是消息没有传达到  %v", uid, r.PostForm.Get("userName"), e))
			}
			Log(fmt.Sprintf("(%d)%s登陆成功", uid, r.PostForm.Get("userName")))
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
			Log(fmt.Sprintf("有人注册但是发生了错误且没有传达 %v", e))
		}
		Log(fmt.Sprintf("有人注册但是发生了错误 %v", err))
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
			Log(fmt.Sprintf("(%d)%s注册账号成功但是没有发送给他这个消息 %v", registeId, r.PostForm.Get("userName"), e))
		}
		Log(fmt.Sprintf("(%d)%s注册了账号", registeId, r.PostForm.Get("userName")))
	}
}
func ChangeUserHeadPortraitBox(c *Client, msgFromUser *MsgFromUser) {
	uid := c.UserInfo.Uid
	now = time.Now()
	nowTime := fmt.Sprintf("%d-%d-%d-%d:%d:%d", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	filename := fmt.Sprintf("Uid%d_%s.%s", uid, nowTime, msgFromUser.UserName[6:])
	index := strings.Index(msgFromUser.Msg, ",")
	base64Data, errDecodeString := base64.StdEncoding.DecodeString(msgFromUser.Msg[index+1:])
	if errDecodeString != nil {
		fmt.Println("errDecodeString", errDecodeString)
	}
	file, err := os.Create("/var/www/html/img/userHeadPortrait/" + filename)
	if err != nil {
		fmt.Println("os.Create", err)
	}
	defer file.Close()
	file.Write(base64Data)
	if errChange := model.ChangeUserHeadPortrait(uid, "https://xxxholic.top/img/userHeadPortrait/"+filename); err != nil {
		fmt.Println("model.ChangeUserHeadPortrait", errChange)
		temp, _ := json.Marshal(MsgFromUser{Status: 311, Msg: fmt.Sprint(errChange)})
		if err != nil {
			fmt.Println("129line", err)
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
			fmt.Println("145line", err)
		}
		Message <- temp
		// time.Sleep(time.Second)
		Log(fmt.Sprintf("(%d)%s修改了头像", c.UserInfo.Uid, c.UserInfo.UserName))
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
			fmt.Println(err)
			temp, _ := json.Marshal(MsgFromUser{Status: 0, Msg: "可能没这个人"})
			c.Socket.WriteMessage(websocket.TextMessage, temp)
			return
		} else {
			msg := msgFromUser.Msg
			//接收者
			temp, err := json.Marshal(MsgFromUser{Status: msgFromUser.Status, Uid: fromId, Msg: msg})
			if err != nil {
				fmt.Println(err)
				return
			}
			if err := ClientMap[toId].Socket.WriteMessage(websocket.TextMessage, temp); err != nil {
				fmt.Println("172lineClientMap[toId].Socket.WriteMessageErr", err)
				return
			}
			//发送者
			temp, _ = json.Marshal(MsgFromUser{Status: (msgFromUser.Status + 1), Uid: toId, Msg: msg})
			if err := c.Socket.WriteMessage(websocket.TextMessage, temp); err != nil {
				fmt.Println("180c.Socket.WriteMessageErr", err)
				return
			}
			Log(fmt.Sprintf("(%d)%s对(%d)%s说:%s", fromId, c.UserInfo.UserName, toId, ClientMap[toId].UserInfo.UserName, msg))
		}
	}

}
