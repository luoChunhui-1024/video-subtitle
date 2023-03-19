package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	"luoChunhui-1024/video-subtitle/videosrt"
	"luoChunhui-1024/video-subtitle/videosrt/tool"
	"net/http"
	"os"
	"path/filepath"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

//func handler(w http.ResponseWriter, r *http.Request) {
//	// 升级HTTP连接为WebSocket连接
//	conn, err := upgrader.Upgrade(w, r, nil)
//	if err != nil {
//		fmt.Println("Upgrade:", err)
//		return
//	}
//	defer conn.Close()
//	fmt.Println("New connection")
//	// 创建百度AI语音识别客户端
//	client := speech.NewSpeechClient("your_app_id", "your_api_key", "your_secret_key")
//	for {
//		// 读取前端发送的音频数据
//		messageType, message, err := conn.ReadMessage()
//		if err != nil {
//			fmt.Println("ReadMessage:", err)
//			break
//		}
//		fmt.Printf("Received message: %d bytes\n", len(message))
//		if messageType == websocket.BinaryMessage {
//			// 识别语音
//			res, err := client.Recognize(message, "pcm", 16000)
//			if err != nil {
//				fmt.Println("Speech recognition error:", err)
//				continue
//			}
//			if len(res.Result) > 0 {
//				text := res.Result[0]
//				fmt.Println("Recognized text:", text)
//				// 发送识别结果
//				err = conn.WriteMessage(websocket.TextMessage, []byte(text))
//				if err != nil {
//					fmt.Println("WriteMessage:", err)
//					break
//				}
//			}
//		} else {
//			fmt.Println("Unknown message type:", messageType)
//		}
//	}
//	fmt.Println("Connection closed")
//}

func RecognizeHandler2(w http.ResponseWriter, r *http.Request) {
	// 升级HTTP连接为WebSocket连接
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Print("升级websocket连接错误:", err)
		return
	}

	defer conn.Close()
	fmt.Println("New connection")
	for {
		// 读取前端发送的音频数据
		messageType, message, err := conn.ReadMessage()
		//_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("ReadMessage:", err)
		}
		fmt.Printf("Received message: %d bytes\n", len(message))

		if messageType == websocket.BinaryMessage {
			// 识别语音
			//fmt.Println("====message", message)
			videoPath, err := convertByteToVideo(message)
			fmt.Println("videoPath = ", videoPath)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Println("====>", err)
				return
			}
			textWord := getSubtitle(videoPath) // 把视频转成音频，把音频传到阿里云 OSS 服务器，利用阿里云语音识别，完成字幕的提取
			fmt.Printf("textWord===> %s\n", textWord)
			if textWord != "" {
				conn.WriteMessage(websocket.TextMessage, []byte(textWord)) // 识别到文本才发送到前端，否则不发送
			}
		} else {
			fmt.Println("Unknown message type:", messageType)
		}
	}
	fmt.Println("Connection closed")
}

func convertByteToVideo(video []byte) (string, error) {
	videoPath := "video.mp4"
	//audioPath := "audio.wav"
	if err := ioutil.WriteFile(videoPath, video, 0644); err != nil {
		log.Println("本地文件===err=", err)
		return "", err
	}
	isExist, err := tool.PathExists(videoPath)
	log.Println("=======> PathExists(videoPath)", isExist, " err=", err)
	return videoPath, nil
}

func getSubtitle(videoPath string) string {
	appDir, err := filepath.Abs(filepath.Dir(os.Args[0])) //应用执行根目录
	if err != nil {
		panic(err)
	}

	//获取应用
	app := videosrt.NewApp(CONFIG)

	appDir = videosrt.WinDir(appDir)

	//初始化应用
	app.Init(appDir)

	//调起应用
	result := app.Run2(videosrt.WinDir(videoPath))

	// 提取文本的文字
	textWord := ""
	for _, itemArray := range result {
		for _, textElement := range itemArray {
			if textElement.Text != "" {
				textWord += textElement.Text + ""
			}
		}
	}
	return textWord
}
