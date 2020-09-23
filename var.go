package viot

import(
	"errors"
	"time"
)


//默认读取数据行大小
const DefaultLineBytes = 1 << 20 // 1 MB

// maxInt32是服务器和传输的字节限制读取器的有效“无限”值。
const maxInt32 int = 1<<31 - 1
var (
  	ErrBodyNotAllowed 	= errors.New("The request method or status code is not allowed")
	ErrGetBodyed		= errors.New("Does not support repeated reading of body")
  	ErrHijacked 		= errors.New("Connection has been hijacked")
  	ErrLaunched			= errors.New("The connection is waiting for the response of the active request")
	ErrAbortHandler 	= errors.New("Abort processing")
	ErrServerClosed 	= errors.New("Server is down")
	ErrDoned			= errors.New("Has been completed")
	ErrConnClose		= errors.New("Device connection is closed")
	ErrReqUnavailable	= errors.New("Request unavailable")
)
var (
	errTooLarge 		= errors.New("Request data is too large")
)


//检测服务器下线时间间隔
var shutdownPollInterval = 500 * time.Millisecond

// aLongTimeAgo是一个非零时间，远在过去，用于立即取消网络操作。
var aLongTimeAgo = time.Unix(1, 0)

//方法集
var methods	= []string{"GET","POST","HEAD","PUT","DELETE","OPTIONS"}




