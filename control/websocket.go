package control

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"model"
	"net/http"
	"strconv"

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
	//ImgData  []byte `json:"imgData"`
}
type MsgToUserOnlie struct {
	Status int              `json:"status"`
	User   []UserSimpleData `json:"user"`
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
		fmt.Println("54line", err)
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
		fmt.Println("78line", err)
	}
	Message <- temp
	ClientMap[c.UserInfo.Uid] = c
	// time.Sleep(time.Second)
	Log(fmt.Sprintf("(%d)%s来到了聊天室", c.UserInfo.Uid, c.UserInfo.UserName))

}

//用户离线
func UserLeave(c *Client) {
	c.Socket.Close()
	close(c.MsgCh)
	delete(ClientMap, c.UserInfo.Uid)
	temp, _ := json.Marshal(MsgFromUser{Status: 100, Msg: fmt.Sprintf("%s已退出聊天室", c.UserInfo.UserName)})
	Message <- temp
	Log(fmt.Sprintf("(%d)%s退出了聊天室", c.UserInfo.Uid, c.UserInfo.UserName))
}
func UserRegister(c *Client) {
	//向其他用户广播该用户上线
	NewUserOnlie(c)
	//向该用户发送在线用户信息
	SendUserOnlieData(c)
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
			case 100:
				msg := msgFromUser.Msg
				temp, _ := json.Marshal(MsgFromUser{Status: 200, Uid: c.UserInfo.Uid, UserName: c.UserInfo.UserName, Msg: msg})
				Log(fmt.Sprintf("(%d)%s:  %s", c.UserInfo.Uid, c.UserInfo.UserName, msg))
				Message <- temp
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
	person, err := model.SelectUserId(cookieUserId.Value)
	if err != nil || err1 != nil || cookieUserId == nil || cookieVerification == nil {
		temp, _ := json.Marshal(MsgFromUser{Status: 30, Msg: "请您先登录"})
		conn.WriteMessage(websocket.TextMessage, temp)
		conn.Close()
		return
	} else if fmt.Sprintf("%x%x", md5.Sum([]byte(person.UserEmail)), md5.Sum([]byte(person.UserPassword))) != cookieVerification.Value {
		temp, _ := json.Marshal(MsgFromUser{Status: 10, Msg: "请您重新登录"})
		conn.WriteMessage(websocket.TextMessage, temp)
		conn.Close()
		return
	} else {
		uid, _ := strconv.Atoi(strconv.FormatInt(person.UserId, 10))
		if ClientMap[uid] != nil {
			temp, _ := json.Marshal(MsgFromUser{Status: 20, Msg: "该账户已登陆"})
			conn.WriteMessage(websocket.TextMessage, temp)
			conn.Close()
			return
		} else {
			client := &Client{
				MsgCh:  make(chan []byte),
				Socket: conn,
				UserInfo: UserSimpleData{
					Uid:              uid,
					UserName:         person.UserName,
					UserHeadPortrait: person.UserHeadPortrait,
				}}
			go UserRegister(client)
		}
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
