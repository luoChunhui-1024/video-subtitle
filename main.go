package main

import (
	"fmt"
	"net/http"
	"os"
)

//定义配置文件
const CONFIG = "config.ini"

func main() {

	//致命错误捕获
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("执行错误 : ", err)
			os.Exit(500)
		}
	}()

	http.HandleFunc("/", RecognizeHandler2)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("ListenAndServe:", err)
	}
}
