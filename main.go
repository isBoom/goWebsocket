package main

import (
	"control"
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("生枝系统start")
	http.HandleFunc(`/login`, control.Login)
	http.HandleFunc(`/registe`, control.Registe)
	http.HandleFunc("/ws", control.Websocket)
	http.ListenAndServeTLS(":8088", "/var/www/html/credentials/xxxholic.top_public.crt", "/var/www/html/credentials/xxxholic.top.key", nil)
}
