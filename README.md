# viot [![Build Status](https://travis-ci.org/456vv/viot.svg?branch=master)](https://travis-ci.org/456vv/viot)
golang viot, 简单的iot服务器。


# **列表：**
```go
const DefaultLineBytes = 1 << 20                                        // 1 MB
var (
    ErrBodyNotAllowed   = verror.TrackError("请求方法或状态码是不允许")
    ErrGetBodyed        = verror.TrackError("不支持重复读取body")
    ErrHijacked         = verror.TrackError("连接已经被劫持")
    ErrLaunched         = verror.TrackError("连接正在等待主动请求的响应")
    ErrAbortHandler     = verror.TrackError("中止处理")
    ErrServerClosed     = verror.TrackError("服务器已经关闭")
    ErrDoned            = verror.TrackError("已经完成")
    ErrConnClose        = verror.TrackError("设备连接已经关闭")
    ErrReqUnavailable   = verror.TrackError("请求不可用")
)
var (
    ServerContextKey = &contextKey{"iot-server"}                        // 服务器
    LocalAddrContextKey = &contextKey{"local-addr"}                     // 监听地址
)

type Handler interface {                                        // 处理函数接口
    ServeIOT(ResponseWriter, *Request)                                  // 处理
}

type HandlerFunc func(ResponseWriter, *Request)                 // 处理函数
    func (T HandlerFunc) ServeIOT(w ResponseWriter, r *Request)         // 函数

type Server struct {                                            // 服务器
    Addr            string                                              // 如果空，TCP监听的地址是，“:http”
    Handler         Handler                                             // 如果nil，处理器调用，http.DefaultServeMux
    ConnState       func(net.Conn, ConnState)                           // 每一个连接跟踪
    ConnHook        func(net.Conn) (net.Conn, error)                    // 连接钩子
    HandlerRequest  func(b io.Reader) (req *Request, err error)         // 处理请求
    HandlerResponse func(b io.Reader) (res *Response, err error)        // 处理响应
    ErrorLog        *log.Logger                                         // 错误？默认是 os.Stderr
    ReadTimeout     time.Duration                                       // 求读取之前，最长期限超时
    WriteTimeout    time.Duration                                       // 响应写入之前，最大持续时间超时
    IdleTimeout     time.Duration                                       // 空闲时间，等待用户重新请求
    TLSNextProto    map[string]func(*Server, *tls.Conn, Handler)        // TLS劫持，["v3"]=function(自身, TLS连接, Handler)
    MaxLineBytes    int                                                 // 限制读取行数据大小
}
    func (T *Server) ListenAndServe() error                     // 监听并服务
    func (T *Server) Serve(l net.Listener) error                        // 服务器监听
    func (T *Server) Close() error                                      // 关闭服务器
    func (T *Server) Shutdown(ctx context.Context) error                // 关闭服务器，等待连接完成
    func (T *Server) RegisterOnShutdown(f func())                       // 注册更新事件
    func (T *Server) SetKeepAlivesEnabled(v bool)                       // 设置长连接开启
type Header map[string]string                                   // 标头
    func (h Header) Set(key, value string)                              // 设置
    func (h Header) Get(key string) string                              // 读取
    func (h Header) Del(key string)                                     // 删除
    func (h Header) clone() Header                                      // 克隆
func ParseIOTVersion(vers string) (major, minor int, ok bool)           // 解析IOT请求版本
func ReadRequest(b io.Reader) (req *Request, err error)                 // 读取请求数据
func ReadResponse(r *bufio.Reader, req *Request) (res *Response, err error) // 读取响应数据
func Nonce() (nonce string, err error)                                  // 生成编号
func Error(w ResponseWriter, err string, code int)                      // 快速设置错误

type RequestIOT struct{                                         // iot接收或发送数据格式
    Nonce   string          `json:"nonce"`//-,omitempty,string,number,boolean
    Proto   string          `json:"proto"`
    Method  string          `json:"method"`
    Path    string          `json:"path"`
    Home    string          `json:"home"`
    Header  Header          `json:"header"`
}
    func (T *RequestIOT) SetBody(i interface{})                             // 设置主体
    func (T *RequestIOT) GetBody() interface{}                              // 读取主体
    func (T *RequestIOT) Marshal() ([]byte, error)                          // 编码
    func (T *RequestIOT) Unmarshal(data []byte) error                       // 解码

type Request struct {                                               // 请求
    nonce       int64                                                       // 编号
    Home        string                                                      // 身份
    Method      string                                                      // 方法
    RequestURI  string                                                      // 请求URL
    URL         *url.URL                                                    // 路径
    Proto       string                                                      // 协议
    ProtoMajor  int                                                         // 协议大版号
    ProtoMinor  int                                                         // 协议小版号
    Header      Header                                                      // 标头
    TLS         *tls.ConnectionState                                        // TLS
    RemoteAddr string                                                       // 远程IP地址
    Close       bool                                                        // 客户要求一次性连接
}

    func (T *Request) GetNonce() string                                     // 读取编号
    func (T *Request) GetBody(i interface{}) error                          // 读取主体
    func (T *Request) SetBody(i interface{}) error                          // 设置主体
    func (T *Request) ProtoAtLeast(major, minor int) bool                   // 判断版本号
    func (T *Request) Context() context.Context                             // 读取上下文
    func (T *Request) WithContext(ctx context.Context) *Request             // 替换上下文
    func (T *Request) GetBasicAuth() (username, password string, ok bool)   // 基本验证
    func (T *Request) SetBasicAuth(username, password string)               // 设置基本验证
    func (T *Request) GetTokenAuth() (token string, ok bool)                // token验证
    func (T *Request) SetTokenAuth(token string)                            // 设置token验证
    func (T *Request) RequestIOT(nonce string) (riot *RequestIOT, err error)// 请求，发往设备的请求
var ErrAbortHandler = errors.New("viot: abort Handler")                     // 错误标头
type ResponseIOT struct{
    Nonce     string                        `json:"nonce"`
    Status    int                            `json:"status"`
    Header     Header                        `json:"header"`
    Body     interface{}                    `json:"body,omitempty"`
}

type Response struct{                                               // 响应
    Status     int                                                          // 状态码
    Header     Header                                                       // 标头
    Body       interface{}                                                  // 主体
    Close      bool                                                         // 服务器关闭连接
    Request    *Request                                                     // 请求
    RemoteAddr string                                                       // 远程IP
}
    func (T *Response) SetNonce(n string)                                   // 读取编号
    func (T *Response) WriteTo(w ResponseWriter)                            // 写入到
    func (T *Response) Write(w io.Writer) error                                // 写入w
    func (T *Response) ResponseIOT(nonce string) (riot *ResponseIOT, err error)// 响应，接收设备的响应
type ResponseWriter interface {                                     // 响应写入接口
    Header() Header                                                         // 标头
    Status(int)                                                             // 状态
    SetBody(interface{}) error                                              // 主体
}
type Hijacker interface {                                           // 劫持接口
    Hijack() (net.Conn, *bufio.ReadWriter, error)                           // 劫持
}
type CloseNotifier interface {                                      // 连接关闭通知接口
    CloseNotify() <-chan bool                                               // 关闭通知
}
type Launcher interface{}{                                            // 发射，服务器使用当前连接作为客户端给智能设置发送信息
    Launch() (tr RoundTripper, err error)                                    // 发射
}
type RoundTripper interface {                                       // 执行一个单一的HTTP事务
    RoundTrip(*Request) (*Response, error)                                  // 单一的HTTP请求
    RoundTripContext(ctx context.Context, req *Request) (resp *Response, err error)    // 单一的HTTP请求(上下文)
}
type Route struct{
    HandlerError    func(w ResponseWriter, r *Request)                        // 处理错误的请求
}
    func (T *Route) HandleFunc(url string,  handler func(w ResponseWriter, r *Request))    // 增加函数
    func (T *Route) ServeIOT(w ResponseWriter, r *Request)                    // 调用函数
```