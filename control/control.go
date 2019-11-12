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

var (
	fs = make(map[int]map[int]int)
)

//登录
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
		if ClientMap[loginInfo.UserId] != nil {
			SendMsg(w, 20, "该账户已登陆")
			model.Log.Info("账号[%d][%s]已登录,有人挤他", loginInfo.UserId, r.PostForm.Get("userName"))
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
				model.Log.Warning("(%d)%s登陆成功但是消息没有传达到  %v", loginInfo.UserId, r.PostForm.Get("userName"), e)
			} else {
				model.Log.Info("(%d)%s登陆成功", loginInfo.UserId, r.PostForm.Get("userName"))
			}
		}
	}

}

//注册
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

//改头
func ChangeUserHeadPortraitBox(c *Client, msgFromUser *MsgFromUser) {
	userId := c.UserInfo.UserId
	now := time.Now()
	nowTime := now.Format("2006-01-02_15:04:05")
	filename := fmt.Sprintf("UserId%d_%s.%s", userId, nowTime, msgFromUser.UserName[6:])
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
	if errChange := model.ChangeUserHeadPortrait(userId, "https://xxxholic.top/img/userHeadPortrait/"+filename); err != nil {
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
			UserId:           c.UserInfo.UserId,
			UserHeadPortrait: msgFromUser.Msg,
		}
		temp, err := json.Marshal(msgToUserOnlie)
		if err != nil {
			model.Log.Warning("json.Marshal %v", err)
		}
		Message <- temp
		model.Log.Info("[%d][%s]修改了头像", c.UserInfo.UserId, c.UserInfo.UserName)
	}
}

//私聊
func PrivateChat(c *Client, msgFromUser *MsgFromUser) {
	fromId := c.UserInfo.UserId
	toId := msgFromUser.UserId
	msg := msgFromUser.Msg
	status := msgFromUser.Status
	//存储离线消息用
	tempSaveFunc := func(isRead int) {
		if err := model.SaveOfflineMessage(fromId, toId, status, msg, isRead); err != nil {
			temp, _ := json.Marshal(MsgFromUser{Status: 0, Msg: "发送离线消息失败"})
			model.Log.Warning("发送离线消息失败 %v", err)
			c.Socket.WriteMessage(websocket.TextMessage, temp)
			return
		}
	}
	//向接收者发送在线消息
	tempSendToFunc := func() {
		//接收者
		temp, _ := json.Marshal(MsgFromUser{Status: status, UserId: fromId, Msg: msg})
		if err := ClientMap[toId].Socket.WriteMessage(websocket.TextMessage, temp); err != nil {
			model.Log.Warning("ClientMap[toId].Socket.WriteMessageErr %v", err)
			return
		}
	}
	//向发送者发送在线消息
	tempSendFromFunc := func() {
		temp, _ := json.Marshal(MsgFromUser{Status: status + 1, UserId: toId, Msg: msg})
		if err := c.Socket.WriteMessage(websocket.TextMessage, temp); err != nil {
			model.Log.Warning("c.Socket.WriteMessageErr", err)
			return
		}
	}

	if mod, err := model.SelectUserId(strconv.Itoa(toId)); err != nil {
		//目标用户无效
		model.Log.Warning("model.SelectUserId %v", err)
		temp, _ := json.Marshal(MsgFromUser{Status: 0, Msg: "可能没这个人"})
		c.Socket.WriteMessage(websocket.TextMessage, temp)
		return
	} else {
		//不在线
		if ClientMap[toId] == nil {
			//从内存判断是否好友
			if fs[fromId][toId] == 1 || fs[toId][fromId] == 1 {
				tempSaveFunc(0)
				tempSendFromFunc()
				model.Log.Info("[离线消息][%d][%s]对[%d][%s]说[%s]", fromId, c.UserInfo.UserName, mod.UserId, mod.UserName, msg)
			} else {
				//从数据库判断是否好友
				if model.IsFriend(fromId, toId) {
					//把是否好友信息记录到内存 下次就不用访问数据库了
					fs[fromId] = make(map[int]int)
					fs[fromId][toId] = 1
					//存储离线消息
					tempSaveFunc(0)
					tempSendFromFunc()
					model.Log.Info("[离线消息][%d][%s]对[%d][%s]说[%s]", fromId, c.UserInfo.UserName, mod.UserId, mod.UserName, msg)
				} else {
					//非好友不允许发送离线消息
					temp, _ := json.Marshal(MsgFromUser{Status: 0, Msg: "对方不在线 只有好友能发送离线消息哟"})
					c.Socket.WriteMessage(websocket.TextMessage, temp)
					return
					model.Log.Info("[离线消息被拒绝][%d][%s]对[%d][%s]说[%s]", fromId, c.UserInfo.UserName, mod.UserId, mod.UserName, msg)
				}
			}
		} else {
			//在线
			if fs[fromId][toId] == 1 || fs[toId][fromId] == 1 {
				tempSaveFunc(1)
				tempSendFromFunc()
				tempSendToFunc()
				model.Log.Info("[%d][%s]对[%d][%s]说[%s]", fromId, c.UserInfo.UserName, mod.UserId, mod.UserName, msg)
			} else {
				if model.IsFriend(fromId, toId) {
					//是好友 消息存到数据库
					fs[fromId] = make(map[int]int)
					fs[fromId][toId] = 1
					tempSaveFunc(1)
					tempSendFromFunc()
					tempSendToFunc()
				} else {
					//发送消息
					tempSendFromFunc()
					tempSendToFunc()
				}
				model.Log.Info("[%d][%s]对[%d][%s]说[%s]", fromId, c.UserInfo.UserName, mod.UserId, mod.UserName, msg)
			}
		}
	}

}

//添加好友
func AddFriendRquest(c *Client, msgFromUser *MsgFromUser) {
	user, err := model.SelectUser(msgFromUser.Msg)
	//莫得这个人
	if err != nil {
		temp, _ := json.Marshal(MsgFromUser{Status: 510, Msg: fmt.Sprintf("没有[%s]这个人啊", msgFromUser.Msg)})
		if err := c.Socket.WriteMessage(websocket.TextMessage, temp); err != nil {
			model.Log.Warning("c.Socket.WriteMessageErr", err)
			return
		}
		model.Log.Info("[%d][%s]向[%s]发送好友请求，但是查无此人", c.UserInfo.UserId, c.UserInfo.UserName, msgFromUser.Msg)
		return
	} else {
		//有了有了
		//-------------- 重复添加
		if c.UserInfo.UserId == user.UserId { //自己加自己
			temp, _ := json.Marshal(MsgFromUser{Status: 500, Msg: "咱能不加自己吗"})
			if err := c.Socket.WriteMessage(websocket.TextMessage, temp); err != nil {
				model.Log.Warning("c.Socket.WriteMessageErr", err)
				return
			}
			model.Log.Info("[%d][%s]非加自己好友", c.UserInfo.UserId, c.UserInfo.UserName)
		} else if model.IsFriend(c.UserInfo.UserId, user.UserId) { //本来就是好友
			temp, _ := json.Marshal(MsgFromUser{Status: 580, Msg: "他已经是你的好友啦，请勿重复添加"})
			if err := c.Socket.WriteMessage(websocket.TextMessage, temp); err != nil {
				model.Log.Warning("c.Socket.WriteMessageErr", err)
				return
			}
			model.Log.Info("[%d][%s]向[%d][%s]发送了重复的好友请求,因为他们本来就是好友", c.UserInfo.UserId, c.UserInfo.UserName, user.UserId, user.UserName)
		} else if err := model.InsertFeiendsRequest(c.UserInfo.UserId, user.UserId); err != nil { //存储好友请求
			model.Log.Warning("InsertFeiendsRequest", err)
			return
		} else {
			temp, _ := json.Marshal(MsgFromUser{Status: 500, Msg: fmt.Sprintf("已向[%s]发送好友请求", msgFromUser.Msg)})
			if err := c.Socket.WriteMessage(websocket.TextMessage, temp); err != nil {
				model.Log.Warning("c.Socket.WriteMessageErr", err)
				return
			}
			model.Log.Info("[%d][%s]向[%d][%s]发送了好友请求", c.UserInfo.UserId, c.UserInfo.UserName, user.UserId, user.UserName)
			//如果此人在线立即发送
			if ClientMap[user.UserId] != nil {
				SendFriendsRequest(ClientMap[user.UserId])
			}
		}

	}
}

//从数据库删除此请求
func DelFriendsRequest(c *Client, msgFromUser *MsgFromUser) {
	model.DelFriendsRequest(msgFromUser.UserId, c.UserInfo.UserId)
}

//添加好友
func AddFriendList(c *Client, msgFromUser *MsgFromUser) {
	//先删库
	model.DelFriendsRequest(msgFromUser.UserId, c.UserInfo.UserId)
	//好友数据库小id在前
	if err := model.AddFriendList(msgFromUser.UserId, c.UserInfo.UserId); err != nil {
		//添加异常
		temp, _ := json.Marshal(MsgFromUser{Status: 570, Msg: fmt.Sprintf("添加好友时遇到了异常")})
		if err := c.Socket.WriteMessage(websocket.TextMessage, temp); err != nil {
			model.Log.Warning("c.Socket.WriteMessageErr", err)
			return
		}
		model.Log.Warning("添加好友时遇到了异常 %v", err)
	} else {
		// c.UserInfo.UserId是接收方
		// msgFromUser.UserId是发送方
		var temp []byte
		var err error
		var msgToUserOnlie = &MsgToUserOnlie{
			Status: 560,
		}
		from, err := model.SelectUserId(strconv.Itoa(msgFromUser.UserId))
		if err != nil {
			model.Log.Debug("%v", err)
			return
		}

		//向接受方发送
		msgToUserOnlie.User = make([]UserSimpleData, 1)
		msgToUserOnlie.User[0] = UserSimpleData{
			UserId:           from.UserId,
			UserHeadPortrait: from.UserHeadPortrait,
			UserName:         from.UserName,
		}
		temp, _ = json.Marshal(msgToUserOnlie)
		if err = c.Socket.WriteMessage(websocket.TextMessage, temp); err != nil {
			model.Log.Warning("c.Socket.WriteMessageErr", err)
			return
		}
		//如果发送方在线 也发送
		if ClientMap[msgFromUser.UserId] != nil {
			msgToUserOnlie.User[0] = UserSimpleData{
				UserId:           c.UserInfo.UserId,
				UserHeadPortrait: c.UserInfo.UserHeadPortrait,
				UserName:         c.UserInfo.UserName,
			}
			temp, _ = json.Marshal(msgToUserOnlie)
			if err := ClientMap[msgFromUser.UserId].Socket.WriteMessage(websocket.TextMessage, temp); err != nil {
				model.Log.Warning("c.Socket.WriteMessageErr", err)
				return
			}
		}
		model.Log.Info("[%d][%s]同意了[%d][%s]的好友请求", c.UserInfo.UserId, c.UserInfo.UserName, from.UserId, from.UserName)
	}

}
