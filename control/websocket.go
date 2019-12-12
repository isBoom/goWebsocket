package control

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"websocket/model"

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
	UserId           int    `json:"userId"`
	UserName         string `json:"userName"`
	UserHeadPortrait string `json:"userHeadPortrait"`
}

type MsgFromUser struct {
	UserId   int    `json:"userId"`
	Status   int    `json:"status"`
	UserName string `json:"userName"`
	Msg      string `json:"msg"`
}
type MsgToUserOnlie struct {
	Status int              `json:"status"`
	Msg    string           `json:"msg"`
	User   []UserSimpleData `json:"user"`
}
type FriendsRequestToUser struct {
	Status         int                 `json:"status"`
	Msg            string              `json:"msg"`
	FriendsRequest []model.FriendsInfo `json:"friendsRequest"`
}
type FriendsListToUser struct {
	Status int                 `json:"status"`
	Msg    string              `json:"msg"`
	User   []model.FriendsInfo `json:"user"`
}
type OfflineMessage struct {
	Status int                    `json:"status"`
	Data   []model.OfflineMessage `json:"data"`
}

//向新登录用户发送在线用户信息
func SendUserOnlieData(c *Client) {

	var msgToUserOnlie = &MsgToUserOnlie{
		Status: 110,
	}
	msgToUserOnlie.User = make([]UserSimpleData, 0)

	for _, data := range ClientMap {
		msgToUserOnlie.User = append(msgToUserOnlie.User, UserSimpleData{
			UserId:           data.UserInfo.UserId,
			UserHeadPortrait: data.UserInfo.UserHeadPortrait,
			UserName:         data.UserInfo.UserName,
		})
	}
	temp, err := json.Marshal(msgToUserOnlie)
	if err != nil {
		model.Log.Warning("json.Marshal %v", err)
		return
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
		UserId:           c.UserInfo.UserId,
		UserHeadPortrait: c.UserInfo.UserHeadPortrait,
		UserName:         c.UserInfo.UserName,
	}
	temp, err := json.Marshal(msgToUserOnlie)
	if err != nil {
		model.Log.Warning("json.Marshal %v", err)
		return
	}
	ClientMap[c.UserInfo.UserId] = c
	Message <- temp
	// time.Sleep(time.Second)
	model.Log.Info("(%d)%s来到了聊天室", c.UserInfo.UserId, c.UserInfo.UserName)

}

//用户离线
func UserLeave(c *Client) {
	c.Socket.Close()
	close(c.MsgCh)
	delete(ClientMap, c.UserInfo.UserId)
	temp, _ := json.Marshal(MsgFromUser{Status: 130, UserId: c.UserInfo.UserId, UserName: c.UserInfo.UserName})
	Message <- temp
	model.Log.Info("(%d)%s退出了聊天室", c.UserInfo.UserId, c.UserInfo.UserName)
}

//发送所有好友列表
func SendFriendsList(c *Client) {
	mod, err := model.SelectFriendslist(c.UserInfo.UserId)
	if err != nil {
		model.Log.Warning("model.SelectFriendslist %v", err)
		return
	} else if len(mod) != 0 {
		friendsListToUser := &FriendsListToUser{
			Status: 540,
		}
		friendsListToUser.User = make([]model.FriendsInfo, 0)
		for _, data := range mod {
			friendsListToUser.User = append(friendsListToUser.User, data)
		}
		temp, _ := json.Marshal(friendsListToUser)
		c.Socket.WriteMessage(websocket.TextMessage, temp)
	}
}

//发送离线消息
func SendOfflineMessage(c *Client) {
	mod, err := model.SelectOfflineMessage(c.UserInfo.UserId)
	if err != nil {
		model.Log.Warning("model.SelectOfflineMessage %v", err)
		return
	} else if len(mod) != 0 {
		temp, _ := json.Marshal(OfflineMessage{Status: 233, Data: mod})
		if err := c.Socket.WriteMessage(websocket.TextMessage, temp); err != nil {
			model.Log.Warning("c.Socket.WriteMessageErr", err)
			return
		}
	}
	//把离线消息置为已读
	if err := model.SetHasRead(c.UserInfo.UserId); err != nil {
		model.Log.Warning("mode.SetHasRead %v", err)
	}
}

//发送请求
func SendFriendsRequest(c *Client) {
	mod, err := model.SelectFriendsRequest(c.UserInfo.UserId)
	if err != nil {
		model.Log.Warning("model.SelectFriendsRequest %v", err)
		return
	} else if len(mod) != 0 {
		friendsRequestToUser := &FriendsRequestToUser{
			Status: 520,
		}
		friendsRequestToUser.FriendsRequest = make([]model.FriendsInfo, 0)
		for _, data := range mod {
			friendsRequestToUser.FriendsRequest = append(friendsRequestToUser.FriendsRequest, data)
		}
		temp, _ := json.Marshal(friendsRequestToUser)
		c.Socket.WriteMessage(websocket.TextMessage, temp)
	}
}
func UserRegister(c *Client) {
	//向其他用户广播该用户上线
	NewUserOnlie(c)
	//向该用户发送在线用户信息
	SendUserOnlieData(c)
	//向该用户发送在好友列表
	SendFriendsList(c)
	//向该用户发送所有好友请求
	SendFriendsRequest(c)
	//两秒后向该用户发送所有离线消息列表 要不然前台渲染不完头像 出现头像不显示bug
	go func() {
		time.Sleep(time.Second)
		SendOfflineMessage(c)
	}()
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
				temp, _ := json.Marshal(MsgFromUser{Status: msgFromUser.Status, UserId: c.UserInfo.UserId, UserName: c.UserInfo.UserName, Msg: msg})
				model.Log.Info("(%d)%s:  %s", c.UserInfo.UserId, c.UserInfo.UserName, msg)
				Message <- temp
			//更改头像
			case 310:
				ChangeUserHeadPortraitBox(c, msgFromUser)
			//私聊 文字/图片
			case 400, 410:
				PrivateChat(c, msgFromUser)
			//请求添加好友
			case 500:
				AddFriendRquest(c, msgFromUser)
			//查看自己的好友请求
			case 530:
				SendFriendsRequest(c)
			//同意加好友
			case 540:
				AddFriendList(c, msgFromUser)
			//拒绝好友关系
			case 550:
				DelFriendsRequest(c, msgFromUser)
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
				UserId:           person.UserId,
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
