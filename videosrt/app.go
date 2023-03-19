package videosrt

import (
	"bytes"
	"fmt"
	"github.com/buger/jsonparser"
	"luoChunhui-1024/video-subtitle/videosrt/aliyun/cloud"
	"luoChunhui-1024/video-subtitle/videosrt/aliyun/oss"
	"luoChunhui-1024/video-subtitle/videosrt/config/ini"
	"luoChunhui-1024/video-subtitle/videosrt/ffmpeg"
	"os"
	"path"
	"strconv"
)

//主应用
type VideoSrt struct {
	Ffmpeg       ffmpeg.Ffmpeg
	AliyunOss    oss.AliyunOss      //oss
	AliyunClound cloud.AliyunClound //语音识别引擎

	IntelligentBlock bool   //智能分段处理
	TempDir          string //临时文件目录
	AppDir           string //应用根目录
}

//获取应用
func NewApp(cfg string) *VideoSrt {
	app := ReadConfig(cfg)

	return app
}

//读取配置
func ReadConfig(cfg string) *VideoSrt {
	if file, e := ini.LoadConfigFile(cfg, "."); e != nil {
		panic(e)
	} else {
		appconfig := &VideoSrt{}

		//AliyunOss
		appconfig.AliyunOss.Endpoint = file.GetMust("aliyunOss.endpoint", "")
		appconfig.AliyunOss.AccessKeyId = file.GetMust("aliyunOss.accessKeyId", "")
		appconfig.AliyunOss.AccessKeySecret = file.GetMust("aliyunOss.accessKeySecret", "")
		appconfig.AliyunOss.BucketName = file.GetMust("aliyunOss.bucketName", "")
		appconfig.AliyunOss.BucketDomain = file.GetMust("aliyunOss.bucketDomain", "")

		//AliyunClound
		appconfig.AliyunClound.AccessKeyId = file.GetMust("aliyunClound.accessKeyId", "")
		appconfig.AliyunClound.AccessKeySecret = file.GetMust("aliyunClound.accessKeySecret", "")
		appconfig.AliyunClound.AppKey = file.GetMust("aliyunClound.appKey", "")

		appconfig.IntelligentBlock = file.GetBoolMust("srt.intelligent_block", false)
		appconfig.TempDir = "temp/audio"

		return appconfig
	}
}

//应用初始化
func (app *VideoSrt) Init(appDir string) {
	app.AppDir = appDir
}

//应用运行
func (app *VideoSrt) Run(video string) {
	if video == "" {
		panic("enter a video file waiting to be processed .")
	}

	//校验视频
	if VaildVideo(video) != true {
		panic("the input video file does not exist .")
	}

	tmpAudioDir := app.AppDir + "/" + app.TempDir
	if !DirExists(tmpAudioDir) {
		//创建目录
		if err := CreateDir(tmpAudioDir, true); err != nil {
			panic(err)
		}
	}
	tmpAudioFile := GetRandomCodeString(15) + ".mp3"
	tmpAudio := tmpAudioDir + "/" + tmpAudioFile

	Log("提取音频文件 ...")

	//分离视频音频
	ExtractVideoAudio(video, tmpAudio)

	Log("上传音频文件 ...")

	//上传音频至OSS
	filelink := UploadAudioToClound(app.AliyunOss, tmpAudio)
	//获取完整链接
	filelink = app.AliyunOss.GetObjectFileUrl(filelink)
	fmt.Println("filelink:", filelink)

	Log("上传文件成功 , 识别中 ...")

	//阿里云录音文件识别
	AudioResult := AliyunAudioRecognition(app.AliyunClound, filelink, app.IntelligentBlock)

	Log("文件识别成功 , 字幕处理中 ...")

	//校验字幕段落是否为空
	if CheckEmptyResult(AudioResult) {
		panic("检测到识别结果为空，生成字幕失败（检查：媒体文件人声是否清晰？语音引擎与媒体语言是否匹配？）")
	}

	//输出字幕文件
	AliyunAudioResultMakeSubtitleFile(video, AudioResult)

	Log("完成")

	//删除临时文件
	if remove := os.Remove(tmpAudio); remove != nil {
		panic(remove)
	}
}

//应用运行
func (app *VideoSrt) Run2(video string) map[int64][]*cloud.AliyunAudioRecognitionResult {
	if video == "" {
		panic("enter a video file waiting to be processed .")
	}

	//校验视频
	if VaildVideo(video) != true {
		panic("the input video file does not exist .")
	}

	tmpAudioDir := app.AppDir + "/" + app.TempDir
	if !DirExists(tmpAudioDir) {
		//创建目录
		if err := CreateDir(tmpAudioDir, true); err != nil {
			panic(err)
		}
	}
	tmpAudioFile := GetRandomCodeString(15) + ".mp3"
	tmpAudio := tmpAudioDir + "/" + tmpAudioFile

	Log("提取音频文件 ...")

	//分离视频音频
	ExtractVideoAudio(video, tmpAudio)

	Log("上传音频文件 ...")

	//上传音频至OSS
	filelink := UploadAudioToClound(app.AliyunOss, tmpAudio)
	//获取完整链接
	filelink = app.AliyunOss.GetObjectFileUrl(filelink)
	fmt.Println("filelink:", filelink)

	Log("上传文件成功 , 识别中 ...")

	//阿里云录音文件识别
	AudioResult := AliyunAudioRecognition(app.AliyunClound, filelink, app.IntelligentBlock)

	Log("文件识别成功 , 字幕处理中 ...")

	//校验字幕段落是否为空
	if CheckEmptyResult(AudioResult) {
		fmt.Println("检测到识别结果为空，生成字幕失败（检查：媒体文件人声是否清晰？）")
	}

	return AudioResult
}

//提取视频音频文件
func ExtractVideoAudio(video string, tmpAudio string) {
	if err := ffmpeg.ExtractAudio(video, tmpAudio); err != nil {
		fmt.Println("提取音频文件失败", tmpAudio, ", err=", err)
	}
}

//上传音频至oss
func UploadAudioToClound(target oss.AliyunOss, audioFile string) string {
	name := ""
	//提取文件名称
	if fileInfo, e := os.Stat(audioFile); e != nil {
		Log("oss 文件名称解析失败 ...", e)
	} else {
		name = fileInfo.Name()
	}

	//上传
	if file, e := target.UploadFile(audioFile, name); e != nil {
		panic(fmt.Sprintf("oss 文件上传失败 %v", e))
	} else {
		return file
	}
}

//阿里云录音文件识别
func AliyunAudioRecognition(engine cloud.AliyunClound, filelink string, intelligent_block bool) (AudioResult map[int64][]*cloud.AliyunAudioRecognitionResult) {
	//创建识别请求
	taskid, client, e := engine.NewAudioFile(filelink)
	if e != nil {
		Log("智能语音接口调用失败 ...", e)
	}

	AudioResult = make(map[int64][]*cloud.AliyunAudioRecognitionResult)

	//遍历获取识别结果
	err := engine.GetAudioFileResult(taskid, client, func(text string) {
		Log(text)
	}, func(result []byte) {
		//mylog.WriteLog( string(result) )

		//结果处理
		statusText, _ := jsonparser.GetString(result, "StatusText") //结果状态
		if statusText == cloud.STATUS_SUCCESS {

			//智能分段
			if intelligent_block {
				cloud.AliyunAudioResultWordHandle(result, func(vresult *cloud.AliyunAudioRecognitionResult) {
					channelId := vresult.ChannelId

					_, isPresent := AudioResult[channelId]
					if isPresent {
						//追加
						AudioResult[channelId] = append(AudioResult[channelId], vresult)
					} else {
						//初始
						AudioResult[channelId] = []*cloud.AliyunAudioRecognitionResult{}
						AudioResult[channelId] = append(AudioResult[channelId], vresult)
					}
				})
				return
			}

			_, err := jsonparser.ArrayEach(result, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
				text, _ := jsonparser.GetString(value, "Text")
				channelId, _ := jsonparser.GetInt(value, "ChannelId")
				beginTime, _ := jsonparser.GetInt(value, "BeginTime")
				endTime, _ := jsonparser.GetInt(value, "EndTime")
				silenceDuration, _ := jsonparser.GetInt(value, "SilenceDuration")
				speechRate, _ := jsonparser.GetInt(value, "SpeechRate")
				emotionValue, _ := jsonparser.GetInt(value, "EmotionValue")

				vresult := &cloud.AliyunAudioRecognitionResult{
					Text:            text,
					ChannelId:       channelId,
					BeginTime:       beginTime,
					EndTime:         endTime,
					SilenceDuration: silenceDuration,
					SpeechRate:      speechRate,
					EmotionValue:    emotionValue,
				}

				_, isPresent := AudioResult[channelId]
				if isPresent {
					//追加
					AudioResult[channelId] = append(AudioResult[channelId], vresult)
				} else {
					//初始
					AudioResult[channelId] = []*cloud.AliyunAudioRecognitionResult{}
					AudioResult[channelId] = append(AudioResult[channelId], vresult)
				}
			}, "Result", "Sentences")
			if err != nil {
				Log("字幕信息解析发生错误 ...", e)
			}
		}
	})
	if err != nil {
		fmt.Println("此次音频文字识别失败，请检查是否为有效音频", err.Error())
	}
	return
}

//阿里云录音识别结果集生成字幕文件
func AliyunAudioResultMakeSubtitleFile(video string, AudioResult map[int64][]*cloud.AliyunAudioRecognitionResult) {
	subfileDir := path.Dir(video)
	subfile := GetFileBaseName(video)

	for _, result := range AudioResult {
		thisfile := subfileDir + "/" + subfile + ".srt"
		//输出字幕文件
		println("输出文件：", thisfile)

		file, e := os.Create(thisfile)
		if e != nil {
			panic(e)
		}

		index := 0
		for _, data := range result {
			linestr := MakeSubtitleText(index, data.BeginTime, data.EndTime, data.Text)
			file.WriteString(linestr)
			index++
		}
		file.Close() //defer
		break
	}
}

//拼接字幕字符串
func MakeSubtitleText(index int, startTime int64, endTime int64, text string) string {
	var content bytes.Buffer
	content.WriteString(strconv.Itoa(index))
	content.WriteString("\n")
	content.WriteString(SubtitleTimeMillisecond(startTime))
	content.WriteString(" --> ")
	content.WriteString(SubtitleTimeMillisecond(endTime))
	content.WriteString("\n")
	content.WriteString(text)
	content.WriteString("\n")
	content.WriteString("\n")
	return content.String()
}

//检查是否为空输出
func CheckEmptyResult(AudioResult map[int64][]*cloud.AliyunAudioRecognitionResult) bool {
	empty := true
	for _, v := range AudioResult {
		for range v {
			empty = false
			break
		}
	}
	return empty
}
