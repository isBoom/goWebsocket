package control

import (
	"crypto/md5"
	"fmt"
	"log"
	"model"
	"net/http"
	"strconv"
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
