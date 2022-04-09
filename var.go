package viot

import (
	"errors"
	"time"

	"github.com/456vv/vweb/v2"
)

// 默认读取数据行大小
const DefaultLineBytes = 1 << 20 // 1 MB

// maxInt32是服务器和传输的字节限制读取器的有效“无限”值。
const maxInt32 int = 1<<31 - 1

var (
	ErrBodyNotAllowed  = errors.New("viot: The request method or status code is not allowed")
	ErrGetBodyed       = errors.New("viot: Does not support repeated reading of body")
	ErrHijacked        = errors.New("viot: Connection has been hijacked")
	ErrLaunched        = errors.New("viot: The connection is waiting for the response of the active request")
	ErrRwaControl      = errors.New("viot: Processing the original data")
	ErrAbortHandler    = errors.New("viot: Abort processing")
	ErrServerClosed    = errors.New("viot: Server is down")
	ErrDoned           = errors.New("viot: Has been completed")
	ErrConnClose       = errors.New("viot: Device connection is closed")
	ErrReqUnavailable  = errors.New("viot: Request unavailable")
	ErrRespUnavailable = errors.New("viot: Response unavailable")

	ErrHostInvalid   = errors.New("host invalid")
	ErrURIInvalid    = errors.New("URI invalid")
	ErrProtoInvalid  = errors.New("proto Invalid")
	ErrMethodInvalid = errors.New("method Invalid")
)

// 检测服务器下线时间间隔
var shutdownPollInterval = 500 * time.Millisecond

// aLongTimeAgo是一个非零时间，远在过去，用于立即取消网络操作。
var aLongTimeAgo = time.Unix(1, 0)

// 方法集
var methods = []string{"GET", "POST", "HEAD", "PUT", "DELETE", "OPTIONS"}

// 请求特征
var reqFeature1 = []string{"\"nonce\"", "\"proto\"", "\"method\"", "\"path\"", "\"home\""} // 过时的，暂时保留
var reqFeature2 = []string{"\"nonce\"", "\"proto\"", "\"method\"", "\"path\"", "\"host\""}

// 响应特征
var respFeature1 = []string{"\"nonce\"", "\"status\""}

type (
	Session   = vweb.Session
	Sessions  = vweb.Sessions
	Globaler  = vweb.Globaler
	Sessioner = vweb.Sessioner
	SiteMan   = vweb.SiteMan
	Site      = vweb.Site
)

// 上下文中使用的key
var (
	ServerContextKey    = &contextKey{"iot-server"} // 服务器
	LocalAddrContextKey = &contextKey{"local-addr"} // 监听地址
	SiteContextKey      = &contextKey{"iot-site"}   // 网站上下文
)
