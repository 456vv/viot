# viot [![Build Status](https://travis-ci.org/456vv/viot.svg?branch=master)](https://travis-ci.org/456vv/viot)
golang viot, 简单的iot服务器。


# **列表：**
```go
const DefaultLineBytes = 1 << 20                                        // 1 MB
var (
    ErrBodyNotAllowed   = errors.New("The request method or status code is not allowed")
    ErrGetBodyed        = errors.New("Does not support repeated reading of body")
    ErrHijacked         = errors.New("Connection has been hijacked")
    ErrLaunched         = errors.New("The connection is waiting for the response of the active request")
    ErrAbortHandler     = errors.New("Abort processing")
    ErrServerClosed     = errors.New("Server is down")
    ErrDoned            = errors.New("Has been completed")
    ErrConnClose        = errors.New("Device connection is closed")
    ErrReqUnavailable   = errors.New("Request unavailable")

	ErrHostInvalid   = errors.New("Host invalid")
	ErrURIInvalid    = errors.New("URI invalid")
	ErrProtoInvalid  = errors.New("Proto Invalid")
	ErrMethodInvalid = errors.New("Method Invalid")
)
var (
    ServerContextKey = &contextKey{"iot-server"}                        // 服务器
    LocalAddrContextKey = &contextKey{"local-addr"}                     // 监听地址
)
var TemplateFunc = vweb.TemplateFunc                                    // 模板函数映射
type Session = vweb.Session                                             // 会话
type Sessions = vweb.Sessions                                           // 会话集
type Globaler = vweb.Globaler                                           // 全局会话
type Sessioner = vweb.Sessioner                                         // 会话接口
type SiteMan = vweb.SiteMan                                             // 站点控制
type Site = vweb.Site                                                   // 站点
type Handler interface {                                        // 处理函数接口
    ServeIOT(ResponseWriter, *Request)                                  // 处理
}

type HandlerFunc func(ResponseWriter, *Request)                 // 处理函数
    func (T HandlerFunc) ServeIOT(w ResponseWriter, r *Request)         // 函数
type LogLevel int                                               // 日志
const (
    LogErr LogLevel    = 1 << iota                              // 错误
    LogDebug                                                    // 调试
)
type Server struct {                                            // 服务器
    Addr            string                                              // 如果空，TCP监听的地址是，“:http”
    Handler         Handler                                             // 如果nil，处理器调用
    BaseContext     func(net.Listener) context.Context                  // 监听上下文
    ConnContext     func(context.Context, net.Conn) (context.Context, net.Conn, error)   // 连接钩子
    ConnState       func(net.Conn, ConnState)                           // 每一个连接跟踪
    HandlerRequest  func(b io.Reader) (req *Request, err error)         // 处理请求
    HandlerResponse func(b io.Reader) (res *Response, err error)        // 处理响应
    ErrorLog        *log.Logger                                         // 错误？默认是 os.Stderr
    ErrorLogLevel   LogLevel                                            // 日志错误级别
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
    func (h Header) Clone() Header                                      // 克隆
func ParseIOTVersion(vers string) (major, minor int, ok bool)           // 解析IOT请求版本
func ReadRequest(b io.Reader) (req *Request, err error)                 // 读取请求数据
func ReadResponse(r io.Reader, req *Request) (res *Response, err error) // 读取响应数据
func Nonce() (nonce string, err error)                                  // 生成编号
func Error(w ResponseWriter, err string, code int)                      // 快速设置错误
type RequestConfig struct{                                         // iot接收或发送数据格式
    Nonce   string          `json:"nonce"`//-,omitempty,string,number,boolean
    Proto   string          `json:"proto"`
    Method  string          `json:"method"`
    Path    string          `json:"path"`
    Host    string          `json:"host"`
    Header  Header          `json:"header"`
}
    func (T *RequestConfig) SetBody(i interface{})                          // 设置主体
    func (T *RequestConfig) GetBody() interface{}                           // 读取主体
    func (T *RequestConfig) Marshal() ([]byte, error)                       // 编码
    func (T *RequestConfig) Unmarshal(data []byte) error                    // 解码
type Request struct {                                               // 请求
    Host        string                                                      // 身份
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
    func (T *Request) RequestConfig(nonce string) (riot *RequestConfig, err error)// 请求，发往设备的请求
type ResponseConfig struct{
    Nonce     string                         `json:"nonce"`
    Status    int                            `json:"status"`
    Header     Header                        `json:"header"`
    Body     interface{}                     `json:"body,omitempty"`
}
type Response struct{                                                   // 响应
    Status     int                                                          // 状态码
    Header     Header                                                       // 标头
    Body       interface{}                                                  // 主体
    Close      bool                                                         // 服务器关闭连接
    Request    *Request                                                     // 请求
    RemoteAddr string                                                       // 远程IP
}
    func (T *Response) SetNonce(n string)                                   // 设置编号
    func (T *Response) WriteAt(w ResponseWriter)                            // 写入到
    func (T *Response) WriteTo(w io.Writer) error                           // 写入w
    func (T *Response) ResponseConfig(nonce string) (riot *ResponseConfig, err error)// 响应，接收设备的响应
type ResponseWriter interface {                                         // 响应写入接口
    Header() Header                                                         // 标头
    Status(int)                                                             // 状态
    SetBody(interface{}) error                                              // 主体
}
type Hijacker interface {                                               // 劫持接口
    Hijack() (net.Conn, *bufio.ReadWriter, error)                           // 劫持
}
type CloseNotifier interface {                                          // 连接关闭通知接口
    CloseNotify() <-chan error                                               // 关闭通知
}
type Launcher interface{}{                                              // 发射，服务器使用当前连接作为客户端给智能设置发送信息
    Launch() RoundTripper                                                   // 发射
}
 type Flusher interface {                                               //缓冲
    Flush()                                                                 // 刷新缓冲
}
type RoundTripper interface {                                           // 执行一个单一的IOT事务
    RoundTrip(*Request) (*Response, error)                                  // 单一的IOT请求
    RoundTripContext(ctx context.Context, req *Request) (resp *Response, err error)    // 单一的IOT请求(上下文)
}
type RawControler interface{                                            //源控制，用于临时处理原始数据
    RawControl(f func(net.Conn, *bufio.Reader) error)                       // 返回错误关闭连接
}
type Route struct{                                                          // 路由
    HandlerError    func(w ResponseWriter, r *Request)                          // 处理错误的请求
}
    func (T *Route) HandleFunc(url string,  handler func(w ResponseWriter, r *Request))    // 增加函数
var DefaultSitePool    = NewSitePool()                                      // 默认站点
type SitePool struct {                                                      // 站点池
    *vweb.SitePool                                                              // 嵌入站点
}
    func NewSitePool() *SitePool                                                // 新建
type TemplateDot struct {                                                   // 模板点
    R        *Request                                                           // 请求
    W        ResponseWriter                                                     // 响应
    Site     *Site                                                              // 站点配置
    Writed   bool                                                               // 表示已经调用写入到客户端。这个是只读的
}
    func (T *TemplateDot) Defer(call interface{}, args ...interface{}) error    // 退同调用
    func (T *TemplateDot) Free()                                                // 释放Defer
    func (T *TemplateDot) Global() Globaler                                     // 全站缓存
    func (T *TemplateDot) Header() Header                                       // 标头
    func (T *TemplateDot) Request() *Request                                    // 请求的信息
    func (T *TemplateDot) ResponseWriter() ResponseWriter                       // 数据写入响应
    func (T *TemplateDot) Launch() RoundTripper                                 // 发射
    func (T *TemplateDot) Hijack() (net.Conn, *bufio.ReadWriter, error)         // 劫持
    func (T *TemplateDot) RootDir(upath string) string                          // 站点的根目录
    func (T *TemplateDot) Session() Sessioner                                   // 用户的会话
    func (T *TemplateDot) Swap() *vmap.Map                                      // 信息交换
    func (T *TemplateDot) Context() context.Context                             // 上下文
    func (T *TemplateDot) WithContext(ctx context.Context)                      // 替换上下文
type TemplateDoter interface {                                              // 模板点
    RootDir(path string) string                                                 // 站点的根目录
    Request() *Request                                                          // 用户的请求信息
    Header() Header                                                             // 标头
    ResponseWriter() ResponseWriter                                             // 数据写入响应
    Launch() RoundTripper                                                       // 发射
    Hijack() (net.Conn, *bufio.ReadWriter, error)                               // 劫持
    Session(token string) Sessioner                                             // 用户的会话缓存
    Global() Globaler                                                           // 全站缓存
    Swap() *vmap.Map                                                            // 信息交换
    Defer(call interface{}, args ... interface{}) error                         // 退回调用
    DotContexter                                                                // 点上下文
}
type DotContexter interface {                                               // 点上下文
    Context() context.Context                                                   // 上下文
    WithContext(ctx context.Context)                                            // 替换上下文
}
type DynamicTemplater interface {                                           // 动态模板
    SetPath(rootPath, pagePath string)                                          // 设置路径
    Parse(r io.Reader) (err error)                                              // 解析
    Execute(out io.Writer, dot interface{}) error                               // 执行
}
type DynamicTemplateFunc func(*ServerHandlerDynamic) DynamicTemplater           // 动态模板方法
type ServerHandlerDynamic struct {                                          // 动态
    //必须的
    RootPath string                                                             // 根目录
    PagePath string                                                             // 主模板文件路径

    //可选的
    Site     *Site                                                              // 站点配置
    Context  context.Context                                                    // 上下文
    Plus     map[string]DynamicTemplateFunc                                     // 支持更动态文件类型
    ReadFile            func(u *url.URL, filePath string) (io.Reader, time.Time, error)     // 读取文件。仅在 .ServeHTTP 方法中使用
    ReplaceParse        func(name string, p []byte) []byte                                  // 替换解析
}
    func (T *ServerHandlerDynamic) Execute(bufw io.Writer, dock interface{}) (err error)        // 执行模板
    func (T *ServerHandlerDynamic) Parse(bufr io.Reader) (err error)                            // 解析模板
    func (T *ServerHandlerDynamic) ParseFile(path string) error                                 // 解析模板文件
    func (T *ServerHandlerDynamic) ParseText(content, name string) error                        // 解析模板文本
    func (T *ServerHandlerDynamic) ServeIOT(rw ResponseWriter, req *Request)                    // 服务IOT
type Client struct{                                                             // 客户端
    Dialer              vconnpool.Dialer                                            // 拨号
    Host                string                                                      // Host
    Addr                string                                                      // 服务器地址
    WriteDeadline       time.Duration                                               // 写入连接超时
    ReadDeadline        time.Duration                                               // 读取连接超时
}
    func (T *Client) Get(url string, header Header) (resp *Response, err error)                             // 快速读取
    func (T *Client) GetCtx(ctx context.Context, urlstr string, header Header) (resp *Response, err error)  // 快速读取（上下文）
    func (T *Client) Do(req *Request) (resp *Response, err error)                                           // 自定义请求
    func (T *Client) DoCtx(ctx context.Context, req *Request)(resp *Response, err error)                    // 自定义请求（上下文）
    func (T *Client) Post(url string, header Header, body interface{}) (resp *Response, err error)          // 快速提交
    func (T *Client) PostCtx(ctx context.Context, urlstr string, header Header, body interface{})           // 快速提交（上下文）

```