package viot

import(
	"github.com/456vv/verror"
	"io"
	"bufio"
	"runtime"
	"fmt"
	//"net/http"
	"context"
	"net"
	"crypto/tls"
	"bytes"
	"sync"
	"sync/atomic"
	"time"
	"golang.org/x/net/http/httpguts"
)

//initNPNRequest==================================================================================================================================
//NPN请求
type initNPNRequest struct {
  	srv *Server				// 上级
  	c *tls.Conn				// 连接
}

//服务接口
func (T initNPNRequest) ServeIOT(rw ResponseWriter, req *Request) {
  	if req.TLS == nil {
  		req.TLS = &tls.ConnectionState{}
  		*req.TLS = T.c.ConnectionState()
  	}
  	if req.RemoteAddr == "" {
  		req.RemoteAddr = T.c.RemoteAddr().String()
  	}
  	if T.srv.Handler != nil {
  		T.srv.Handler.ServeIOT(rw, req)
  	}
}


//连接
type conn struct {
	server *Server								// 上级，服务器
  	rwc net.Conn								// 上级，原始连接
  	ctx			context.Context					// 上下文
	cancelCtx 	context.CancelFunc				// 取消上下文
	remoteAddr string							// 远程IP
	tlsState *tls.ConnectionState				// TLS状态
	werr error									// 写错误
	r *connReader								// 读取
	bufr *bufio.Reader							// 读缓冲
	bufw *bufio.Writer							// 写缓冲
	curReq atomic.Value							// 当前请求
	curRes atomic.Value							// 当前响应
	curState atomic.Value						// 当前的连接状态
	mu sync.Mutex								// 锁
	hijackedv atomicBool						// 劫持
	launchedv atomicBool						// 发射
	activeReq 	map[string]chan *Response		// 主动请求
	//passiveReq
	closed		bool							// 关闭
}


//劫持连接
func (T *conn) hijackLocked() (rwc net.Conn, buf *bufio.ReadWriter, err error) {
  	T.mu.Lock()
	defer T.mu.Unlock()

  	//判断主动请求
  	if T.launchedv.isTrue() {
  		return nil, nil, ErrLaunched
  	}
  	//判断劫持
  	if T.hijackedv.setTrue() {
  		return nil, nil, ErrHijacked
  	}
	
  	//结束后台接收数据
  	T.r.abortPendingRead()
  
  	T.rwc.SetDeadline(time.Time{})
  
  	buf = bufio.NewReadWriter(T.bufr, bufio.NewWriter(T.rwc))
  	
  	//后台收到一个字节数据
  	if T.r.hasByte {
  		if _, err := T.bufr.Peek(T.bufr.Buffered() + 1); err != nil {
  			return nil, nil, verror.TrackErrorf("意外的Peek失败读取缓冲的字节: %v", err)
  		}
  	}
  	T.setState(StateHijacked)
	
	//回收缓冲对象
  	putBufioWriter(T.bufw)
  	T.bufw = nil
  	return
}


func (T *conn) inLaunch() bool {
  	T.mu.Lock()
  	defer T.mu.Unlock()
	return len(T.activeReq) != 0
}


//发射
func (T *conn) RoundTrip(req *Request) (resp *Response, err error){
	return T.RoundTripContext(context.Background(), req)
}

func (T *conn) RoundTripContext(ctx context.Context, req *Request) (resp *Response, err error){
  	T.mu.Lock()
  	defer T.mu.Unlock()
	
  	if T.closed {
  		return nil, ErrConnClose
  	}
  	
  	if T.hijackedv.isTrue() {
  		return nil, ErrHijacked
  	}
  	
 	//请求不能为空
	if req == nil {
		return nil, ErrReqUnavailable
	}
	
  	T.launchedv.setTrue()

	nonce, err := Nonce()
	if err != nil {
		return nil, err
	}
	
	//导出设备支持的请求格式
	riot, err := req.RequestIOT(nonce)
	if err != nil {
		return nil, err
	}
	
	//设备支持的请求格式转字节
	reqByte, err := riot.Marshal()
	if err != nil {
		return nil, err
	}

	if T.activeReq == nil {
  		T.activeReq = make(map[string]chan *Response)
  	}
	done := make(chan *Response)
	T.activeReq[nonce]=done
  	defer func(){
  		delete(T.activeReq, nonce)
  		close(done)
	}()
	
	//设置写入超时
	if d := T.server.WriteTimeout; d != 0 {
  		T.rwc.SetWriteDeadline(time.Now().Add(d))
  	}

	n, err := T.bufw.Write(reqByte)
	if err != nil {
		return nil, err
	}
	if rbn := len(reqByte); n != rbn {
		return nil, verror.TrackErrorf("实据数据长度 %d，已发送数据长度 %d", rbn, n)
	}
	T.bufw.Flush()
	T.mu.Unlock()
	defer T.mu.Lock()
	select{
	case <- ctx.Done():
		return nil, ctx.Err()
	case <- T.ctx.Done():
		return nil, verror.TrackErrorf("连接已经关闭: %v", T.ctx.Err())
	case res := <- done:
		res.Request = req
		return res, nil
	}
}

func (T *conn) readLineBytes() (b []byte, err error) {

  	if d := T.server.ReadTimeout; d != 0 {
  		T.rwc.SetReadDeadline(time.Now().Add(time.Duration(d) * time.Millisecond))
  	}

  	//设置读取限制大小
  	T.r.setReadLimit(T.server.maxLineBytes())
	//恢复读取大小限制
	defer T.r.setInfiniteReadLimit()

  	//读取行格式
	tp := newTextprotoReader(T.bufr)
  	b, err = tp.ReadLineBytes()
  	defer putTextprotoReader(tp)
  	if err != nil {
		//读取数据超出限制
		if T.r.hitReadLimit() {
			return nil, errTooLarge
		}
  		return nil, err
  	}
	return b, err
}

func (T *conn) readResponse(ctx context.Context, br io.Reader) (res *Response, err error) {
	if T.hijackedv.isTrue() {
		return nil, ErrHijacked
	}

	//使用外部解析函数
	if hr := T.server.HandlerResponse; hr != nil {
		res, err = hr(br)
	}else{
		res, err = readResponse(br)
	}
	if err != nil {
		return
	}
	res.RemoteAddr	= T.remoteAddr
	return
}


  
//解析请求
func (T *conn) readRequest(ctx context.Context, br io.Reader) (req *Request, err error) {
	if T.hijackedv.isTrue() {
		return nil, ErrHijacked
	}
  	
	//使用外部解析函数
	if hr := T.server.HandlerRequest; hr != nil {
		req, err = hr(br)
	}else{
		req, err = readRequest(br)
	}
	if err != nil {
		return nil, err
	}
	
	if req.ProtoMajor != 1 {
		return nil, verror.TrackError("不受支持的协议版本")
	}
	
	if req.Home != "" && !httpguts.ValidHostHeader(req.Home) {
		return nil, verror.TrackError("畸形的 Home")
	}
	
	
	req.ctx, req.cancelCtx = context.WithCancel(ctx)
	req.RemoteAddr	= T.remoteAddr
	req.TLS			= T.tlsState
	return
}

	

//服务
func (T *conn) serve(ctx context.Context) {
	T.remoteAddr = T.rwc.RemoteAddr().String()
	ctx = context.WithValue(ctx, LocalAddrContextKey, T.rwc.LocalAddr())
	defer func(){
		if err := recover(); err != nil && err != ErrAbortHandler {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			T.server.logf("viot: 服务器意外错误 %v: %v\n%s", T.remoteAddr, err, buf)
		}
		if !T.hijackedv.isTrue() {
			T.server.logf("viot: 远程IP(%s)断开网络\n", T.remoteAddr)
			T.Close()
		}
	}()
	T.server.logf("viot: 远程IP(%s)连接网络\n", T.remoteAddr)

//	if tlsConn, ok := T.rwc.(*tls.Conn); ok {
//		//这里等验证，暂时用不上
//		//意思就是暂时不支持TLS
//		if d := T.server.ReadTimeout; d != 0 {
//			T.rwc.SetReadDeadline(time.Now().Add(d))
//		}
//		if d := T.server.WriteTimeout; d != 0 {
//			T.rwc.SetWriteDeadline(time.Now().Add(d))
//		}
//		if err := tlsConn.Handshake(); err != nil {
//			T.server.logf("viot: TLS handshake error from %s: %v", T.rwc.RemoteAddr(), err)
//			return
//		}
//		T.tlsState = new(tls.ConnectionState)
//		*T.tlsState = tlsConn.ConnectionState()
//		//待验证证书请求的协议
//		if proto := T.tlsState.NegotiatedProtocol; validNPN(proto) {
//			if fn := T.server.TLSNextProto[proto]; fn != nil {
//				h := initNPNRequest{T.server, tlsConn}
//				fn(T.server, tlsConn, h)
//			}
//			return
//		}
//	}
	
	//JSON格式
	
	T.r 	= &connReader{conn:T}
	T.bufr 	= newBufioReader(T.r)
	T.bufw 	= newBufioWriterSize(checkConnErrorWriter{T}, 4<<10)
	
	
	T.ctx, T.cancelCtx = context.WithCancel(ctx)
	defer T.cancelCtx()
	for {
		if !T.server.doKeepAlives() {
			//我们正处于关机模式。 
			return
		}
		lineBytes, err := T.readLineBytes()
		if err != nil {
			if isCommonNetReadError(err) {
				return
			}
			T.server.logf(verror.TrackErrorf("从IP(%v)接收到数据读取\"行\"发生错误（%v）", T.remoteAddr, err).Error())
			return
		}
		//T.server.logf("viot: 从远程IP(%s)读取数据行line:\n%s\n\n", T.remoteAddr, lineBytes)
		
		//空行跳过
		if len(lineBytes) == 0 {
			continue
		}
		br 		:= bytes.NewReader(lineBytes)
		isReq 	:= bytes.Contains(lineBytes, []byte("\"proto\":"))
		
		//主动向设备发送请求
		//等待设备响应信息
		if !isReq {
			if T.inLaunch() {
				res, err := T.readResponse(T.ctx, br)
				if err != nil {
					T.server.logf(verror.TrackErrorf("从IP(%v)接收到数据读取\"响应\"发生错误（%v）", T.remoteAddr, err).Error())
					return
				}
				T.setState(StateActive)
				
				T.mu.Lock()
				if cres, ok := T.activeReq[res.nonce]; ok {
					select {
					case cres <- res:
						delete(T.activeReq, res.nonce)
					}
				}
				T.mu.Unlock()
				
				if T.hijackedv.isTrue() {
					return
				}
				T.setState(StateIdle)
				if err = T.idleWait(); err != nil {
					//等待数据，读取超时就退出
					return
				}
				//没有错误，重新读取
				continue
			}
			//收到错误格式的数据，退出
			return
		}
		
		//被动得到设备发来请求
		//等待服务器响应信息
		req, err := T.readRequest(T.ctx, br)
		if err != nil {
			fmt.Fprintf(T.rwc, `{"body":%q,"header":{"Connection": "close"},"nonce":"-1","status":400}`,"Bad Request: "+err.Error())
			T.closeWriteAndWait()
			return
		}

		T.setState(StateActive)
		w := &responseWrite{
			conn			: T,
			req				: req,
			header			: make(Header),
			closeNotifyCh	: make(chan bool, 1),
		}
		w.cw.res = w // w.cw = chunkWriter{w}
		T.curReq.Store(req)
		T.curRes.Store(w)
				
		//后台接收数据
		T.r.startBackgroundRead()
		
		//设置写入超时时间
		if d := T.server.WriteTimeout; d != 0 {
	  		T.rwc.SetWriteDeadline(time.Now().Add(d))
	  	}

		//这里内部不能 go func 和 ctx 一起使用。否则会被取消
		serverHandler{T.server}.ServeIOT(w, w.req)
		w.req.cancelCtx()
		
		T.curReq.Store((*Request)(nil))
		T.curRes.Store((*responseWrite)(nil))

		if T.hijackedv.isTrue() {
			return
		}
		
		//设置完成，生成body，发送至客户端
		w.done()
		
		//停止后台接收数据
		T.r.abortPendingRead()
		
		//不能重用连接，客户端 或 服务端设置了不支持重用
		if w.closeAfterReply  || T.werr != nil {
			T.closeWriteAndWait()
			return
		}
		
		T.setState(StateIdle)
		
		if err = T.idleWait(); err != nil {
			//等待数据，读取超时就退出
			return
		}

	}
}

//设置连接状态
func (T *conn) setState(state ConnState) {
	switch state {
	case StateNew:
		T.server.trackConn(T, true)
	case StateHijacked, StateClosed:
		T.server.trackConn(T, false)
	}
	T.curState.Store(connStateInterface[state])
	if hook := T.server.ConnState; hook != nil {
		hook(T.rwc, state)
	}
}

type ConnState int
const (
	StateNew ConnState = iota
	StateActive
	StateIdle
	StateHijacked
	StateClosed
)

var stateName = map[ConnState]string{
  	StateNew:      "new",
  	StateActive:   "active",
  	StateIdle:     "idle",
  	StateHijacked: "hijacked",
  	StateClosed:   "closed",
}

func (c ConnState) String() string {
	return stateName[c]
}
var connStateInterface = [...]interface{}{
	StateNew:      StateNew,
	StateActive:   StateActive,
	StateIdle:     StateIdle,
	StateHijacked: StateHijacked,
	StateClosed:   StateClosed,
}

func (T *conn) idleWait() error {
	if d := T.server.idleTimeout(); d != 0 {
		T.rwc.SetReadDeadline(time.Now().Add(d))
		if _, err := T.bufr.Peek(4); err != nil {
			return err
		}
	}
	T.rwc.SetReadDeadline(time.Time{})
	return nil
}

//回收缓冲对象
func (T *conn) finalFlush() {
	//如果连接是被劫持，不支持调用此函数，否则爆panic
	if T.bufr != nil {
		putBufioReader(T.bufr)
		T.bufr=nil
	}
	if T.bufw != nil {
		T.bufw.Flush()
		putBufioWriter(T.bufw)
		T.bufw = nil
	}
}


// rstAvoidanceDelay是在关闭整个套接字之前关闭TCP连接的写入端之后我们休眠的时间量。 
// 通过睡眠，我们增加了客户端看到我们的FIN并处理其最终数据的机会，然后再处理后续的RS，从而关闭已知未读数据的连接。 
// 这个RST似乎主要发生在BSD系统上。 （和Windows？）这个超时有点武断（周围的延迟）。
const rstAvoidanceDelay = 500 * time.Millisecond
type closeWriter interface {
  	CloseWrite() error
}
 var _ closeWriter = (*net.TCPConn)(nil)
 
 //关闭并写入
func (T *conn) closeWriteAndWait() {
	if tcp, ok := T.rwc.(closeWriter); ok {
		tcp.CloseWrite()
	}
	time.Sleep(rstAvoidanceDelay)
}


//关闭连接
func (T *conn) Close() {
	//取消连接上下文
	T.cancelCtx()
	
	//需要上锁，否则会清空T.bufr 或 T.bufw
  	T.mu.Lock()
	defer T.mu.Unlock()
	
	T.closed = true
	//释放缓冲对象
	T.finalFlush()
 	T.rwc.Close()
 	T.setState(StateClosed)
}

//checkConnErrorWriter==================================================================================================================================
//检查写入错误
type checkConnErrorWriter struct {
	c *conn																	// 上级
}
//写入
func (T checkConnErrorWriter) Write(p []byte) (n int, err error) {
	n, err = T.c.rwc.Write(p)
	if err != nil && T.c.werr == nil {
		T.c.werr = err
		T.c.server.logf("viot: 远程IP(%s)数据写入失败error(%s)，预写入data(%s)\n", T.c.remoteAddr, err, p)
		T.c.cancelCtx()	//取消当前连接的上下文
	}
	return
}


//connReader==================================================================================================================================
type connReader struct {
  	conn *conn																// 上级
  	                            											
  	mu      sync.Mutex														// 锁
  	hasByte bool															// 检测有数据
  	byteBuf [1]byte															// 第一个数据，检测时候得到一个数据
  	cond    *sync.Cond														// 组
  	inRead  bool															// 正在读取
  	aborted bool															// 结束
  	remain  int																// 剩下
}

//锁，条件等待
func (T *connReader) lock() {
	T.mu.Lock()
	if T.cond == nil {
		T.cond = sync.NewCond(&T.mu)
	}
}

//解锁
func (T *connReader) unlock() {T.mu.Unlock()}
//设置读取限制
func (T *connReader) setReadLimit(remain int) 	{ T.remain = remain }
//恢复默认读取限制
func (T *connReader) setInfiniteReadLimit()		{ T.remain = maxInt32 }
//超出读取限制
func (T *connReader) hitReadLimit() bool        { return T.remain <= 0 }
//开始后台读取
func (T *connReader) startBackgroundRead() {
	T.lock()
  	defer T.unlock()
  	if T.inRead {
  		panic("invalid concurrent Body.Read call")
  	}
  	if T.hasByte {
  		return
  	}
  	T.inRead = true
  	T.conn.rwc.SetReadDeadline(time.Time{})
  	go T.backgroundRead()
}
//后台读取
func (T *connReader) backgroundRead() {
	n, err := T.conn.rwc.Read(T.byteBuf[:])
	T.lock()
	if n == 1 {
		T.hasByte = true
		T.closeNotify() //请求第一条时候，还没处理。又接收到一请求，则发出取消信息
	}
	if ne, ok := err.(net.Error); ok && T.aborted && ne.Timeout() {
		//忽略这个错误。 这是另一个调用abortPendingRead的例程的预期错误。
	}else if err != nil {
		//主动关闭连接，造成读取失败
		T.handleReadError(err)
	}
	T.aborted = false
	T.inRead = false
	T.unlock()
	T.cond.Broadcast()
	
}
//中止等待读取
func (T *connReader) abortPendingRead() {
	T.lock()
	defer T.unlock()
	if !T.inRead {
		return
	}
	T.aborted = true
	T.conn.rwc.SetReadDeadline(aLongTimeAgo)
	for T.inRead {
		T.cond.Wait()
	}
	T.conn.rwc.SetReadDeadline(time.Time{})
}

//读取错误，需要取消所有下上文
func (T *connReader) handleReadError(err error) {
	T.conn.cancelCtx()  // 上下文取消
	T.closeNotify()		// 信息通道取消
}

//连接关闭通知
func (T *connReader) closeNotify() {
	res, _ := T.conn.curRes.Load().(*responseWrite)
	if res != nil {
		res.closeNotify()
	}
}

//读取数据
func (T *connReader) Read(p []byte) (n int, err error) {
	T.lock()
	if T.inRead {
		T.unlock()
		panic("当前调用 .Read() 无效")
	}
	if T.hitReadLimit() {
		T.unlock()
		return 0, io.EOF
	}
	if len(p) == 0 {
		T.unlock()
		return 0, nil
	}
	if len(p) > T.remain {
		p = p[:T.remain]
	}
	if T.hasByte {
		p[0] = T.byteBuf[0]
		T.hasByte = false
		T.unlock()
		return 1, nil
	}
	T.inRead = true
	T.unlock()
	n, err = T.conn.rwc.Read(p)
	T.lock()
	T.inRead = false
	if err != nil {
		//如果数据到了结尾io.EOF，发出读取错误
		T.handleReadError(err)
	}
	T.remain -=n
	T.unlock()
	T.cond.Broadcast()
	return n, err
}




















































