package cloud

import (
	"encoding/json"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/pkg/errors"
	"strconv"
	"time"
)

//SDK
//https://help.aliyun.com/document_detail/94072.html?spm=a2c4g.11186623.6.584.3a1153d5yDFr5B

type AliyunClound struct {
	AccessKeyId     string
	AccessKeySecret string
	AppKey          string
}

//阿里云录音文件识别结果集
type AliyunAudioRecognitionResult struct {
	Text            string //文本结果
	ChannelId       int64  //音轨ID
	BeginTime       int64  //该句的起始时间偏移，单位为毫秒
	EndTime         int64  //该句的结束时间偏移，单位为毫秒
	SilenceDuration int64  //本句与上一句之间的静音时长，单位为秒
	SpeechRate      int64  //本句的平均语速，单位为每分钟字数
	EmotionValue    int64  //情绪能量值1-10，值越高情绪越强烈
}

//阿里云识别词语数据集
type AliyunAudioWord struct {
	Word      string
	ChannelId int64
	BeginTime int64
	EndTime   int64
}

// 地域ID，常量内容，请勿改变
const REGION_ID string = "cn-shanghai"
const ENDPOINT_NAME string = "cn-shanghai"
const PRODUCT string = "nls-filetrans"
const DOMAIN string = "filetrans.cn-shanghai.aliyuncs.com"
const API_VERSION string = "2018-08-17"
const POST_REQUEST_ACTION string = "SubmitTask"
const GET_REQUEST_ACTION string = "GetTaskResult"

// 请求参数key
const KEY_APP_KEY string = "appkey"
const KEY_FILE_LINK string = "file_link"
const KEY_VERSION string = "version"
const KEY_ENABLE_WORDS string = "enable_words"

//是否打开ITN，中文数字将转为阿拉伯数字输出，默认值为false
const KEY_ENABLE_INVERSE_TEXT_NORMAL = "enable_inverse_text_normalization"

//是否启⽤语义断句，取值：true/false，默认值false
const KEY_ENABLE_SEMANTIC_SENTENCE_DETECTION = "enable_semantic_sentence_detection"

//是否启用时间戳校准功能，取值：true/false，默认值false
const KEY_ENABLE_TIMESTAMP_ALIGNMENT = "enable_timestamp_alignment"

// 响应参数key
const KEY_TASK string = "Task"
const KEY_TASK_ID string = "TaskId"
const KEY_STATUS_TEXT string = "StatusText"
const KEY_RESULT string = "Result"

// 状态值
const STATUS_SUCCESS string = "SUCCESS"
const STATUS_RUNNING string = "RUNNING"
const STATUS_QUEUEING string = "QUEUEING"
const SUCCESS_WITH_NO_VALID_FRAGMENT string = "SUCCESS_WITH_NO_VALID_FRAGMENT" // 阿里云识别为无效录音

//发起录音文件识别
//接口文档 https://help.aliyun.com/document_detail/90727.html?spm=a2c4g.11186623.6.581.691af6ebYsUkd1
func (c AliyunClound) NewAudioFile(fileLink string) (string, *sdk.Client, error) {
	client, err := sdk.NewClientWithAccessKey(REGION_ID, c.AccessKeyId, c.AccessKeySecret)
	if err != nil {
		return "", client, err
	}

	postRequest := requests.NewCommonRequest()
	postRequest.Domain = DOMAIN
	postRequest.Version = API_VERSION
	postRequest.Product = PRODUCT
	postRequest.ApiName = POST_REQUEST_ACTION
	postRequest.Method = "POST"

	mapTask := make(map[string]string)
	mapTask[KEY_APP_KEY] = c.AppKey
	mapTask[KEY_FILE_LINK] = fileLink
	// 新接入请使用4.0版本，已接入(默认2.0)如需维持现状，请注释掉该参数设置
	mapTask[KEY_VERSION] = "4.0"
	// 设置是否输出词信息，默认为false，开启时需要设置version为4.0
	mapTask[KEY_ENABLE_WORDS] = "true"

	//统一后处理
	mapTask[KEY_ENABLE_INVERSE_TEXT_NORMAL] = "true"
	mapTask[KEY_ENABLE_SEMANTIC_SENTENCE_DETECTION] = "true"
	mapTask[KEY_ENABLE_TIMESTAMP_ALIGNMENT] = "true"

	// to json
	task, err := json.Marshal(mapTask)
	if err != nil {
		return "", client, errors.New("to json error .")
	}
	postRequest.FormParams[KEY_TASK] = string(task)
	// 发起请求
	postResponse, err := client.ProcessCommonRequest(postRequest)
	if err != nil {
		return "", client, err
	}
	postResponseContent := postResponse.GetHttpContentString()
	//校验请求
	if postResponse.GetHttpStatus() != 200 {
		return "", client, errors.New("录音文件识别请求失败 , Http错误码 : " + strconv.Itoa(postResponse.GetHttpStatus()))
	}
	//解析数据
	var postMapResult map[string]interface{}
	err = json.Unmarshal([]byte(postResponseContent), &postMapResult)
	if err != nil {
		return "", client, errors.New("to map struct error .")
	}

	var taskId = ""
	var statusText = ""
	statusText = postMapResult[KEY_STATUS_TEXT].(string)

	//检验结果
	if statusText == STATUS_SUCCESS {
		taskId = postMapResult[KEY_TASK_ID].(string)
		return taskId, client, nil
	}

	return "", client, errors.New("录音文件识别请求失败 !")
}

//获取录音文件识别结果
//接口文档 https://help.aliyun.com/document_detail/90727.html?spm=a2c4g.11186623.6.581.691af6ebYsUkd1
func (c AliyunClound) GetAudioFileResult(taskId string, client *sdk.Client, logOutput func(text string), callback func(result []byte)) (err error) {
	getRequest := requests.NewCommonRequest()
	getRequest.Domain = DOMAIN
	getRequest.Version = API_VERSION
	getRequest.Product = PRODUCT
	getRequest.ApiName = GET_REQUEST_ACTION
	getRequest.Method = "GET"
	getRequest.QueryParams[KEY_TASK_ID] = taskId
	statusText := ""

	var (
		trys               = 0
		getResponse        *responses.CommonResponse
		getResponseContent string
	)
	//遍历获取识别结果
	for trys < 10 {

		if trys != 0 {
			logOutput("尝试重新查询识别结果，第" + strconv.Itoa(trys) + "次")
		}

		getResponse, err = client.ProcessCommonRequest(getRequest)
		if err != nil {
			logOutput("查询识别结果失败：" + err.Error())
			trys++
			time.Sleep(time.Second * time.Duration(trys))
			continue
		}

		getResponseContent = getResponse.GetHttpContentString()
		if getResponse.GetHttpStatus() != 200 {
			logOutput("查询识别结果失败，Http错误码：" + strconv.Itoa(getResponse.GetHttpStatus()))
			trys++
			time.Sleep(time.Second * time.Duration(trys))
			continue
		}

		var getMapResult map[string]interface{}
		err = json.Unmarshal([]byte(getResponseContent), &getMapResult)
		if err != nil {
			trys++
			logOutput("查询识别结果失败，解析结果失败：" + err.Error())
			continue
		}

		//校验遍历条件
		statusText = getMapResult[KEY_STATUS_TEXT].(string)
		if statusText == STATUS_RUNNING || statusText == STATUS_QUEUEING {
			time.Sleep(3 * time.Second)
		} else {
			break
		}
	}

	if statusText == STATUS_SUCCESS && getResponse != nil {
		//调用回调函数
		callback(getResponse.GetHttpContentBytes())
	} else {
		err = errors.New("录音文件识别失败 , (" + c.GetErrorStatusTextMessage(statusText) + ")")
		return
	}
	return
}

//获取错误信息
func (c AliyunClound) GetErrorStatusTextMessage(statusText string) string {
	var code map[string]string = map[string]string{
		"REQUEST_APPKEY_UNREGISTERED":    "阿里云智能语音项目未创建/无访问权限。请检查语音引擎Appkey是否填写错误；如果是海外地区，在软件创建语音引擎时，服务区域需要选择“海外”",
		"USER_BIZDURATION_QUOTA_EXCEED":  "单日2小时识别免费额度超出限制",
		"FILE_DOWNLOAD_FAILED":           "文件访问失败，请检查OSS存储空间访问权限。请将OSS存储空间设置为“公共读”",
		"FILE_TOO_LARGE":                 "音频文件超出512MB",
		"FILE_PARSE_FAILED":              "音频文件解析失败，请检查音频文件是否有损坏",
		"UNSUPPORTED_SAMPLE_RATE":        "采样率不匹配",
		"FILE_TRANS_TASK_EXPIRED":        "音频文件识别任务过期，请重试",
		"REQUEST_INVALID_FILE_URL_VALUE": "音频文件访问失败，请检查OSS存储空间访问权限",
		"FILE_404_NOT_FOUND":             "音频文件访问失败，请检查OSS存储空间访问权限404",
		"FILE_403_FORBIDDEN":             "音频文件访问失败，请检查OSS存储空间访问权限403",
		"FILE_SERVER_ERROR":              "音频文件访问失败，请检查请求的文件所在的服务是否可用",
		"INTERNAL_ERROR":                 "识别内部通用错误，请稍候重试",
		"SUCCESS_WITH_NO_VALID_FRAGMENT": "识别结果查询接口调用成功，但是没有识别到有效语音。可检查录音文件是否有有效语音。",
	}

	if _, ok := code[statusText]; ok {
		return code[statusText]
	} else {
		return statusText
	}
}
