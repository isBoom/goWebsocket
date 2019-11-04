package main

import (
	"control"
	"fmt"
	"model"
	"net/http"
)

func main() {
	model.LogLever = "debug"
	fmt.Println("生枝系统start")
	http.HandleFunc(`/login`, control.Login)
	http.HandleFunc(`/registe`, control.Registe)
	http.HandleFunc("/ws", control.Websocket)
	http.ListenAndServeTLS(":8088", "/credentials/xxxholic.top_public.crt", "/credentials/xxxholic.top.key", nil)
}
