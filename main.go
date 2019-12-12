package main

import (
	"fmt"
	"net/http"
	"websocket/control"
	"websocket/model"
)

func main() {
	model.LogLever = "debug"
	fmt.Println("生枝系统start")
	http.HandleFunc(`/login`, control.Login)
	http.HandleFunc(`/registe`, control.Registe)
	http.HandleFunc("/ws", control.Websocket)
	http.ListenAndServeTLS(":8088", "/credentials/xxxholic.top_public.crt", "/credentials/xxxholic.top.key", nil)
}
