package model

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var (
	DB *sqlx.DB
)

func init() {
	var err error
	DB, err = sqlx.Connect("mysql", "root:root@tcp(39.106.169.153:3306)/sky?charset=utf8&&parseTime=true")
	if err != nil {
		fmt.Println("sqlx.Connect_______", err)
	}
}

type User struct {
	UserId           int64  `json:"userId" db:"userId"`
	UserName         string `json:"userName" db:"userName"`
	UserPassword     string `json:"userPassword" db:"userPassword"`
	UserEmail        string `json:"userEmail" db:"userEmail"`
	UserCreateDate   string `json:"userCreateDate" db:"userCreateDate"`
	UserHeadPortrait string `json:"userHeadPortrait" db:"userHeadPortrait"`
}

func LoginInfo(userName string, userPassword string) (*User, error) {
	var err error
	mod := &User{}
	err = DB.Get(mod, "select * from userInfo where (userName=? or userEmail=?) and userPassword=? limit 1", userName, userName, userPassword)
	if err != nil {
		err = errors.New("密码错误")
		return mod, err
	}
	return mod, err
}

func SelectUserId(userId string) (*User, error) {
	var err error
	mod := &User{}
	err = DB.Get(mod, "select * from userInfo where userId=? limit 1", userId)
	if err != nil {
		err = errors.New("该用户不存在")
		return mod, err
	}
	return mod, err
}
func RegisteUser(userName string, userPassword string, userEmail string) (int64, error) {
	mod := &User{}
	err := DB.Get(mod, "select * from userInfo where userEmail=? limit 1", userEmail)
	if err == nil {
		err = errors.New("该邮箱已存在")
		return 0, err
	}
	err = DB.Get(mod, "select * from userInfo where userName=? limit 1", userName)
	if err == nil {
		err = errors.New("该昵称已存在")
		return 0, err
	}
	now := time.Now()
	userHeadPortrait := fmt.Sprintf("https://xxxholic.top/img/userHeadPortrait/%d.jpg", rand.Intn(13))
	userCreateDate := fmt.Sprintf("%d-%d-%d %d:%d:%d", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	rand.Int()
	res, err0 := DB.Exec("insert into userInfo(userName,userPassword,userEmail,userCreateDate,userHeadPortrait) VALUES(?,?,?,?,?)", userName, userPassword, userEmail, userCreateDate, userHeadPortrait)
	if err0 != nil {
		err0 = errors.New("添加用户失败错误")
		return 0, err0
	}
	id, err0 := res.LastInsertId()
	if err0 != nil {
		err0 = errors.New("获取用户id失败")
		return 0, err0
	}
	return id, nil

}
func ChangeUserHeadPortrait(userId int, userHeadPortrait string) error {
	mod := &User{}
	err := DB.Get(mod, "select * from userId where userId=? limit 1", userId)
	if err == nil {
		err = errors.New("该用户不存在")
		return err
	}
	_, err0 := DB.Exec("UPDATE userInfo SET userHeadPortrait = ? where userId=?", userHeadPortrait, userId)
	if err0 != nil {
		err0 = errors.New("更改头像失败")
		return err0
	}
	return nil

}

// func UserAll() ([]User, error) {
// 	mods := make([]User, 0)
// 	err := DB.Select(&mods, "select * from `userinfo`")
// 	return mods, err
// }
