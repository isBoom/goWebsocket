package model

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"
	"websocket/mylogger"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var (
	DB       *sqlx.DB
	Log      = mylogger.NewLog(LogLever)
	LogLever string
)

func init() {
	var err error
	DB, err = sqlx.Connect("mysql", "root:root@tcp(39.106.169.153:3306)/sky?charset=utf8&&parseTime=true")
	if err != nil {
		Log.Fatal("数据库连接失败 %v", err)
	}
}

type User struct {
	UserId           int    `json:"userId" db:"userId"`
	UserName         string `json:"userName" db:"userName"`
	UserPassword     string `json:"userPassword" db:"userPassword"`
	UserEmail        string `json:"userEmail" db:"userEmail"`
	UserCreateDate   string `json:"userCreateDate" db:"userCreateDate"`
	UserHeadPortrait string `json:"userHeadPortrait" db:"userHeadPortrait"`
}
type FriendsInfo struct {
	UserId           int    `json:"userId" db:"userId"`
	UserName         string `json:"userName" db:"userName"`
	UserHeadPortrait string `json:"userHeadPortrait" db:"userHeadPortrait"`
}
type OfflineMessage struct {
	Id     int    `json:"id" db:"id"`
	FromId int    `json:"fromId" db:"fromId"`
	ToId   int    `json:"toId" db:"toId"`
	Status int    `json:"status" db:"status"`
	Msg    string `json:"msg" db:"msg"`
	TimeD  int    `json:"timeD" db:"timeD"`
	TimeS  string `json:"timeS" db:"timeS"`
	IsRead int    `json:"isRead" db:"isRead"`
}
type FriendsList struct {
	Id      int `json:"id" db:"id"`
	FriendA int `json:"friendA" db:"friendA"`
	FriendB int `json:"friendB" db:"friendB"`
}

func LoginInfo(userName string, userPassword string) (*User, error) {
	var err error
	mod := &User{}
	err = DB.Get(mod, "select * from userInfo where userName=? or userEmail=? limit 1", userName, userName)
	if err != nil {
		return mod, errors.New("帐号不存在")
	}
	err = DB.Get(mod, "select * from userInfo where (userName=? or userEmail=?) and userPassword=? limit 1", userName, userName, userPassword)
	if err != nil {
		return mod, errors.New("账号或密码错误")
	}
	return mod, nil
}

func SelectUserId(userId string) (*User, error) {
	var err error
	mod := &User{}
	if err = DB.Get(mod, "select * from userInfo where userId=? limit 1", userId); err != nil {
		return mod, errors.New("该用户不存在")
	}
	return mod, err
}
func SelectUser(msg string) (*User, error) {
	var err error
	mod := &User{}
	if err = DB.Get(mod, "select * from userInfo where userName=? or userEmail=? limit 1", msg, msg); err != nil {
		Log.Debug("%v", err)
		return mod, errors.New("该用户不存在")
	}
	return mod, nil
}
func RegisteUser(userName string, userPassword string, userEmail string) (int, error) {
	var err error
	mod := &User{}
	if err := DB.Get(mod, "select * from userInfo where userEmail=? limit 1", userEmail); err == nil {
		err = errors.New("该邮箱已存在")
		return 0, err
	}
	if err = DB.Get(mod, "select * from userInfo where userName=? limit 1", userName); err == nil {
		err = errors.New("该昵称已存在")
		return 0, err
	}
	now := time.Now()
	userHeadPortrait := fmt.Sprintf("https://xxxholic.top/img/userHeadPortrait/%d.jpg", rand.Intn(13))
	userCreateDate := now.Format("2006-01-02_15:04:05")
	res, err0 := DB.Exec("insert into userInfo(userName,userPassword,userEmail,userCreateDate,userHeadPortrait) VALUES(?,?,?,?,?)", userName, userPassword, userEmail, userCreateDate, userHeadPortrait)
	if err0 != nil {
		err0 = errors.New("添加用户失败错误")
		return 0, err0
	}
	tempId, err1 := res.LastInsertId()
	if err1 != nil {
		err1 = errors.New("获取用户id失败")
		return 0, err1
	}
	id := int(tempId)
	return id, nil
}
func ChangeUserHeadPortrait(userId int, userHeadPortrait string) error {
	var err error
	mod := &User{}
	if err = DB.Get(mod, "select * from userInfo where userId=? limit 1", userId); err != nil {
		err = errors.New("该用户不存在")
		return err
	}
	if _, err0 := DB.Exec("UPDATE userInfo SET userHeadPortrait = ? where userId=?", userHeadPortrait, userId); err != nil {
		fmt.Println(err0)
		return err0
	}
	return nil

}
func InsertFeiendsRequest(fromId, toId int) error {
	var err error
	mod := &FriendsInfo{}
	if err = DB.Get(mod, "select fromId as userId from friendsRequest where fromId=? and toId=? limit 1", fromId, toId); err == nil {
		//该请求数据库里已存在 不重复添加 但依旧向发起人回复发送成功
		return nil
	} else if _, err = DB.Exec("insert into friendsRequest(fromId,toId) VALUES(?,?)", fromId, toId); err != nil {
		return err
	}
	return nil
}
func SelectFriendsRequest(id int) ([]FriendsInfo, error) {
	var err error
	mod := make([]FriendsInfo, 0)
	if err = DB.Select(&mod, "select userInfo.userId,userInfo.userName from friendsRequest left join userInfo on friendsRequest.fromId=userInfo.userId  where friendsRequest.toId=?", id); err != nil {
		return nil, err
	}
	return mod, nil
}
func DelFriendsRequest(fromId, toId int) {
	DB.Exec("delete from friendsRequest where fromId = ? and toId = ? ", fromId, toId)
}
func IsFriend(fromId, toId int) bool {
	mod := make([]FriendsList, 0)
	if fromId > toId {
		fromId, toId = toId, fromId
	}
	if err := DB.Select(&mod, "select * from friends where friendA=? and friendB=?", fromId, toId); err != nil {
		Log.Debug("%v", err)
		return false
	} else if len(mod) == 0 {
		return false
	}
	return true
}
func AddFriendList(smallId, bigId int) error {
	var err error
	if smallId > bigId {
		smallId, bigId = bigId, smallId
	}
	if _, err = DB.Exec("insert into friends(friendA,friendB) VALUES(?,?)", smallId, bigId); err != nil {
		return err
	}
	return nil
}

func SelectFriendslist(id int) ([]FriendsInfo, error) {
	var err error
	mod := make([]FriendsInfo, 0)
	if err = DB.Select(&mod, "SELECT userId ,userName,userHeadPortrait from friends INNER JOIN userInfo on (friendA=? and userInfo.userId=friendB) or (friendB=? and userInfo.userId=friendA)", id, id); err != nil {
		return nil, err
	}
	return mod, nil
}
func SaveOfflineMessage(fromId, toId, status int, msg string, isRead int) error {
	//timeD为int 比较方便
	now := time.Now()
	timeD, _ := strconv.Atoi(now.Format("20060102"))
	timeS := now.Format("15:04:05")
	var err error
	if _, err = DB.Exec("INSERT INTO offlineMessage(fromId,toId,status,msg,timeD,timeS,isRead) VALUES(?,?,?,?,?,?,?)", fromId, toId, status, msg, timeD, timeS, isRead); err != nil {
		return err
	}
	return nil
}
func SelectOfflineMessage(id int) ([]OfflineMessage, error) {
	var err error
	//timeD为int 比较方便
	now := time.Now()
	timeD, _ := strconv.Atoi(now.AddDate(0, 0, -3).Format("20060102"))
	mod := make([]OfflineMessage, 0)
	if err = DB.Select(&mod, "SELECT fromId ,toId,status,msg from offlineMessage where (fromId=? or toId =?) and (timeD > ? or isRead=0)", id, id, timeD); err != nil {
		return nil, err
	}
	return mod, nil
}
func SetHasRead(id int) error {
	if _, err := DB.Exec("UPDATE offlineMessage SET isRead = 1 where  toId =?", id); err != nil {
		return err
	}
	return nil
}
