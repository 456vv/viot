package viot

import(
	"errors"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
	"crypto/tls"
	"context"
	"io"
	"fmt"
)

//上下文中使用的key
var (
	ServerContextKey = &contextKey{"iot-server"}						// 服务器
	LocalAddrContextKey = &contextKey{"local-addr"}						// 监听地址
)

//处理函数接口
type Handler interface {
  	ServeIOT(ResponseWriter, *Request)
}

//处理函数
type HandlerFunc func(ResponseWriter, *Request)
func (T HandlerFunc) ServeIOT(w ResponseWriter, r *Request) {
  	T(w, r)
}

//服务处理函数，在服务器没有设置Handler字段，为了保证不出错。
type serverHandler struct {
  	srv *Server															//服务器
}
//处理函数
func (T serverHandler) ServeIOT(rw ResponseWriter, req *Request) {
  	handler := T.srv.Handler
  	if handler == nil {
  		//这个要做一个默认处理！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！！
  		return
  	}
  	handler.ServeIOT(rw, req)
 }

type LogLevel int
const (
    LogErr LogLevel    = 1 << iota		//1
    LogDebug							//2
)

//服务器
type Server struct { 
    Addr            string                                              // 如果空，TCP监听的地址是，“:8000”
    Handler         Handler                                             // 如果nil，处理器调用，http.DefaultServeMux
	BaseContext 	func(net.Listener) context.Context					// 监听上下文
    ConnContext     func(context.Context, net.Conn) (context.Context, net.Conn, error)   // 连接钩子
    ConnState       func(net.Conn, ConnState)                           // 每一个连接跟踪
    HandlerRequest  func(b io.Reader) (req *Request, err error)     	// 处理请求
    HandlerResponse	func(b io.Reader) (res *Response, err error)		// 处理响应
    ErrorLog        *log.Logger                                         // 错误？默认是 os.Stderr
    ErrorLogLevel	LogLevel											// 日志错误级别
    ReadTimeout     time.Duration                                       // 求读取之前，最长期限超时
    WriteTimeout    time.Duration                                       // 响应写入之前，最大持续时间超时
    IdleTimeout     time.Duration                                       // 空闲时间，等待用户重新请求
    TLSNextProto    map[string]func(*Server, *tls.Conn, Handler)        // TLS劫持，["v3"]=function(自身, TLS连接, Handler)
    MaxLineBytes    int                                                 // 限制读取行数据大小
    disableKeepAlives int32                                             // 禁止长连接
    inShutdown        int32                                             // 判断服务器是否已经下线
    


    mu          sync.Mutex                                              // 锁
    listeners   map[*net.Listener]struct{}                              // 监听集
    activeConn  map[*conn]struct{}                                      // 连接集
    doneChan    chan struct{}                                           // 服务关闭
    onShutdown  []func()                                                // 服务器下线事件
}

//初始化
func (T *Server) init() {
	if T.doneChan == nil {
		T.doneChan = make(chan struct{})
	}
}

//行数据大小
func (T *Server) maxLineBytes() int {
  	if T.MaxLineBytes > 0 {
  		return T.MaxLineBytes
  	}
  	return DefaultLineBytes
}

//关闭通道
func (T *Server) closeDoneChan() {
	select {
	case <-T.doneChan:
		//如果已经关闭，不需要再关闭，直接跳过
	default:
		close(T.doneChan)
	}
}

//记录监听
func (T *Server) trackListener(ln *net.Listener, add bool) bool {
	T.mu.Lock()
	defer T.mu.Unlock()
	
	if T.listeners == nil {
		T.listeners = make(map[*net.Listener]struct{})
	}
	if add {
		if T.shuttingDown() {
			return false
		}
		T.listeners[ln]=struct{}{}
	}else{
		
		delete(T.listeners, ln)
		
		if len(T.listeners) == 0 && len(T.activeConn) == 0 {
  			T.doneChan = nil
  		}
	}
	return true
}

//删除监听
func (T *Server) closeListeners() error {
	T.mu.Lock()
	defer T.mu.Unlock()
	var err error
	for ln := range T.listeners {
		if cerr := (*ln).Close(); cerr != nil && err == nil {
			err = cerr
		}
		delete(T.listeners, ln)
	}
	return err
}

//记录连接
func (T *Server) trackConn(c *conn, add bool) {
	T.mu.Lock()
	defer T.mu.Unlock()
	if T.activeConn == nil {
		T.activeConn = make(map[*conn]struct{})
	}
	if add {
		T.activeConn[c]=struct{}{}
	}else{
		delete(T.activeConn, c)
	}
}

//关闭连接
func (T *Server) closeConns() error {
	T.mu.Lock()
	defer T.mu.Unlock()
	for c := range T.activeConn {
		c.rwc.Close()
		delete(T.activeConn, c)
	}
	return nil
}

//服务器监听，监听地址可以设置Addr。默认为""，则是8000
//	error			错误
func (T *Server) ListenAndServe() error {
	addr := T.Addr
	if addr == "" {
		addr = ":8000"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return T.Serve(ln)
}

//服务器监听
//	l net.Listener	监听
//	error			错误
func (T *Server) Serve(l net.Listener) error {
	origListener := l
	l = &tcpKeepAliveListener{l.(*net.TCPListener)}
	defer l.Close()
	
	if !T.trackListener(&l, true) {
		//服务器下线
		return ErrServerClosed
	}
	defer T.trackListener(&l, false)
	
	T.init()
	
	baseCtx := context.Background()
	if T.BaseContext != nil {
		baseCtx = T.BaseContext(origListener)
		if baseCtx == nil {
			return errors.New("viot: BaseContext returned a nil context")
		}
	}
	
	ctx := context.WithValue(baseCtx, ServerContextKey, T)
	var tempDelay time.Duration
	for {
		rw, e := l.Accept()
		if e != nil {
			select {
			case <-T.doneChan:
				//服务器关闭后，信道被打通。退出
				return ErrServerClosed
			default:
			}
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				tempDelay = delay(tempDelay, time.Second)
				continue
			}
			return e
		}
		tempDelay = 0
		
		//新 goroutine 进程
		go func(ctx context.Context, rw net.Conn){
			nrw := rw
			connCtx := ctx
			var err error
			if T.ConnContext != nil {
				connCtx, nrw, err = T.ConnContext(ctx, rw)
				if err != nil {
					defer rw.Close()
					T.logf(LogErr, err.Error())
					return
				}
			}
			c := &conn{server: T, rwc: nrw}
			c.setState(StateNew)
			c.serve(connCtx)
		}(ctx, rw)
	}
}

//关闭服务器
//	error			错误
func (T *Server) Close() error {	
	//关闭服务器
	T.closeDoneChan()
	
	//关闭监听和连接
	T.closeConns()
	return T.closeListeners()
}

//空闲超时时间，如果没有设置，则使用读取时间
func (T *Server) idleTimeout() time.Duration {
	if T.IdleTimeout != 0 {
		return T.IdleTimeout
	}
	return T.ReadTimeout
}

	
//关闭，等待连接完成
//	ctx context.Context	上下文
//	error				错误
func (T *Server) Shutdown(ctx context.Context) error {
	atomic.AddInt32(&T.inShutdown, 1)
  	defer atomic.AddInt32(&T.inShutdown, -1)
  	
  	T.closeDoneChan()
  	lnerr := T.closeListeners()
	for _, f := range T.onShutdown {
  		go f()
  	}
  	
  	//定时关闭空闲连接
  	ticker := time.NewTicker(shutdownPollInterval)
  	defer ticker.Stop()
  	for {
  		//返回 false 表示还有连接不是空闲状态
  		if T.closeIdleConns() {
  			return lnerr
  		}
  		select {
  		case <- ctx.Done():
  			return ctx.Err()
  		case <-ticker.C:
  		}
  	}
}

//注册更新事件
//	f func()		服务下线时调用此函数
func (T *Server) RegisterOnShutdown(f func()) {
  	T.onShutdown = append(T.onShutdown, f)
}

//设置长连接开启
//	v bool			设置支持长连接
func (T *Server) SetKeepAlivesEnabled(v bool) {
	if v {
		atomic.StoreInt32(&T.disableKeepAlives, 0)
		return
	}
	atomic.StoreInt32(&T.disableKeepAlives, 1)
	
	//关闭空闲的连接，让新连接生效keep-Alives
	T.closeIdleConns()
	
}

//日志
func (T *Server) logf(level LogLevel, format string, v ...interface{}) error {
    if T.ErrorLog != nil && T.ErrorLogLevel >= level {
		err := fmt.Errorf(format+"\n", v...)
		T.ErrorLog.Output(2, err.Error())
		return err
    }
    return nil
}

//判断服务器是否支持长连接
func (T *Server) doKeepAlives() bool {
  	return atomic.LoadInt32(&T.disableKeepAlives) == 0 && !T.shuttingDown()
}

//判断服务器下线...
func (T *Server) shuttingDown() bool {
  	return atomic.LoadInt32(&T.inShutdown) != 0
}

//关闭空闲连接
func (T *Server) closeIdleConns() bool {
	T.mu.Lock()
	defer T.mu.Unlock()
  	quiescent := true
	for c := range T.activeConn {
		cs, ok := c.curState.Load().(ConnState)
		if !ok || cs != StateIdle {
			quiescent = false
			continue
		}
		c.rwc.Close()
		delete(T.activeConn, c)
	}
	//如果没有可用的空闲连接，返回true
	return quiescent
}
