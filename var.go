package viot

import(
	"github.com/456vv/verror"
	"time"
)


//默认读取数据行大小
const DefaultLineBytes = 1 << 20 // 1 MB

// maxInt32是服务器和传输的字节限制读取器的有效“无限”值。
const maxInt32 int = 1<<31 - 1

var (
  	ErrBodyNotAllowed 	= verror.TrackError("请求方法或状态码是不允许")
	ErrGetBodyed		= verror.TrackError("不支持重复读取body")
  	ErrHijacked 		= verror.TrackError("连接已经被劫持")
  	ErrLaunched			= verror.TrackError("连接正在等待主动请求的响应")
	ErrAbortHandler 	= verror.TrackError("中止处理")
	ErrServerClosed 	= verror.TrackError("服务器已经关闭")
	ErrDoned			= verror.TrackError("已经完成")
	ErrConnClose		= verror.TrackError("设备连接已经关闭")
	ErrReqUnavailable	= verror.TrackError("请求不可用")
)
var (
	errTooLarge 		= verror.TrackError("要求数据过大")
)


//检测服务器下线时间间隔
var shutdownPollInterval = 500 * time.Millisecond

// aLongTimeAgo是一个非零时间，远在过去，用于立即取消网络操作。
var aLongTimeAgo = time.Unix(1, 0)

//方法集
var methods	= []string{"GET","POST","HEAD","PUT","DELETE","OPTIONS"}




