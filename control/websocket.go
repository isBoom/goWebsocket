package control

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"model"
	"net/http"

	"github.com/gorilla/websocket"
)

var (
	ClientMap map[int]*Client = make(map[int]*Client)
	Message                   = make(chan []byte)
)

type Client struct {
	MsgCh    chan []byte     `json:"msgCh"`
	Socket   *websocket.Conn `json:"socket"`
	UserInfo UserSimpleData  `json:"userInfo"`
}
type UserSimpleData struct {
	Uid              int    `json:"uid"`
	UserName         string `json:"userName"`
	UserHeadPortrait string `json:"userHeadPortrait"`
}

type MsgFromUser struct {
	Uid      int    `json:"uid"`
	Status   int    `json:"status"`
	UserName string `json:"userName"`
	Msg      string `json:"msg"`
}
type MsgToUserOnlie struct {
	Status int              `json:"status"`
	Msg    string           `json:"msg"`
	User   []UserSimpleData `json:"user"`
}
type FriendsrRquestToUser struct {
	Status         int                    `json:"status"`
	Msg            string                 `json:"msg"`
	FriendsrRquest []model.FriendsrRquest `json:"friendsrRquest"`
}

//向新登录用户发送在线用户信息
func SendUserOnlieData(c *Client) {

	var msgToUserOnlie = &MsgToUserOnlie{
		Status: 110,
	}
	msgToUserOnlie.User = make([]UserSimpleData, 0)

	for _, data := range ClientMap {
		msgToUserOnlie.User = append(msgToUserOnlie.User, UserSimpleData{
			Uid:              data.UserInfo.Uid,
			UserHeadPortrait: data.UserInfo.UserHeadPortrait,
			UserName:         data.UserInfo.UserName,
		})
	}
	temp, err := json.Marshal(msgToUserOnlie)
	if err != nil {
		model.Log.Warning("json.Marshal %v", err)
	}
	c.Socket.WriteMessage(websocket.TextMessage, temp)

}

//新用户上线
func NewUserOnlie(c *Client) {
	var msgToUserOnlie = &MsgToUserOnlie{
		Status: 120,
	}
	msgToUserOnlie.User = make([]UserSimpleData, 1)
	msgToUserOnlie.User[0] = UserSimpleData{
		Uid:              c.UserInfo.Uid,
		UserHeadPortrait: c.UserInfo.UserHeadPortrait,
		UserName:         c.UserInfo.UserName,
	}
	temp, err := json.Marshal(msgToUserOnlie)
	if err != nil {
		model.Log.Warning("json.Marshal %v", err)
	}
	ClientMap[c.UserInfo.Uid] = c
	Message <- temp
	// time.Sleep(time.Second)
	model.Log.Info("(%d)%s来到了聊天室", c.UserInfo.Uid, c.UserInfo.UserName)

}

//用户离线
func UserLeave(c *Client) {
	c.Socket.Close()
	close(c.MsgCh)
	delete(ClientMap, c.UserInfo.Uid)
	temp, _ := json.Marshal(MsgFromUser{Status: 130, Uid: c.UserInfo.Uid, UserName: c.UserInfo.UserName})
	Message <- temp
	model.Log.Info("(%d)%s退出了聊天室", c.UserInfo.Uid, c.UserInfo.UserName)
}

//发送请求
func SendFriendsrRquest(c *Client) {
	mod, err := model.SelectFriendsrRquest(c.UserInfo.Uid)
	if err != nil {
		model.Log.Warning("model.SelectFriendsrRquest%v", err)
	}
	temp, _ := json.Marshal(FriendsrRquestToUser{Status: 520, FriendsrRquest: mod})
	c.Socket.WriteMessage(websocket.TextMessage, temp)
}
func UserRegister(c *Client) {
	//向其他用户广播该用户上线
	NewUserOnlie(c)
	//向该用户发送在线用户信息
	SendUserOnlieData(c)
	//向该用户发送在好友列表

	//向该用户发送所有好友请求
	SendFriendsrRquest(c)
	//向该用户发送所有离线消息列表

	//read
	go func() {
		msgFromUser := &MsgFromUser{}
		for {
			err := c.Socket.ReadJSON(msgFromUser)
			if err != nil {
				//用户离线
				UserLeave(c)
				return
			}
			switch msgFromUser.Status {
			case 200, 210: //普通群聊 文字/图片
				msg := msgFromUser.Msg
				temp, _ := json.Marshal(MsgFromUser{Status: msgFromUser.Status, Uid: c.UserInfo.Uid, UserName: c.UserInfo.UserName, Msg: msg})
				model.Log.Info("(%d)%s:  %s", c.UserInfo.Uid, c.UserInfo.UserName, msg)
				Message <- temp
			case 310:
				//更改头像
				ChangeUserHeadPortraitBox(c, msgFromUser)
			case 400, 410: //私聊 文字/图片
				PrivateChat(c, msgFromUser)
			case 500: //请求添加好友
				AddFriend(c, msgFromUser)
			}
		}
	}()
	//write
	for {
		if err := c.Socket.WriteMessage(websocket.TextMessage, <-c.MsgCh); err != nil {
			return
		}
	}
}
func Websocket(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	//将http协议升级成websocket协议
	conn, err := (&websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}).Upgrade(res, req, nil)
	if err != nil {
		http.NotFound(res, req)
		return
	}
	//验证cookie登录
	cookieUserId, err := req.Cookie("userId")
	cookieVerification, err1 := req.Cookie("verification")
	if err != nil || err1 != nil { //没有cookie
		temp, _ := json.Marshal(MsgFromUser{Status: 30, Msg: "请您先登录"})
		conn.WriteMessage(websocket.TextMessage, temp)
		conn.Close()
		return
	} else if person, err2 := model.SelectUserId(cookieUserId.Value); err2 != nil { //cookie不正确
		temp, _ := json.Marshal(MsgFromUser{Status: 30, Msg: "请您先登录"})
		conn.WriteMessage(websocket.TextMessage, temp)
		conn.Close()
		return
	} else if fmt.Sprintf("%x%x", md5.Sum([]byte(person.UserEmail)), md5.Sum([]byte(person.UserPassword))) != cookieVerification.Value { //cookie不正确
		temp, _ := json.Marshal(MsgFromUser{Status: 10, Msg: "请您重新登录"})
		conn.WriteMessage(websocket.TextMessage, temp)
		conn.Close()
		return
	} else if ClientMap[person.UserId] != nil { //重复登陆
		temp, _ := json.Marshal(MsgFromUser{Status: 20, Msg: "该账户已登陆"})
		conn.WriteMessage(websocket.TextMessage, temp)
		conn.Close()
		return
	} else { //登陆成功 创建用户
		client := &Client{
			MsgCh:  make(chan []byte),
			Socket: conn,
			UserInfo: UserSimpleData{
				Uid:              person.UserId,
				UserName:         person.UserName,
				UserHeadPortrait: person.UserHeadPortrait,
			}}
		go UserRegister(client)
	}

}
func init() {
	go func() {
		for {
			conn := <-Message
			for _, c := range ClientMap {
				c.MsgCh <- conn
			}
		}
	}()
}
