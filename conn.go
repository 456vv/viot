package viot

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/456vv/vconn"
	"golang.org/x/net/http/httpguts"
)

// initNPNRequest==================================================================================================================================
// NPN请求
type initNPNRequest struct {
	ctx context.Context // 上下文
	srv *Server         // 上级
	c   *tls.Conn       // 连接
}

func (T initNPNRequest) BaseContext() context.Context { return T.ctx }

// 服务接口
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

// 连接
type conn struct {
	server           *Server                   // 上级，服务器
	rwc              net.Conn                  // 上级，原始连接
	ctx              context.Context           // 上下文
	cancelCtx        context.CancelFunc        // 取消上下文
	remoteAddr       string                    // IP
	tlsState         *tls.ConnectionState      // TLS状态
	vc               *vconn.Conn               // 读取
	bufr             *bufio.Reader             // 读缓冲
	bufw             *bufio.Writer             // 写缓冲
	curState         atomic.Value              // 当前的连接状态
	mu               sync.Mutex                // 锁
	hijackedv        atomicBool                // 劫持
	activeReq        map[string]chan *Response // 主动请求
	closed           bool                      // 关闭
	handleFunc       func(net.Conn, *bufio.Reader) error
	parser           Parser
	parserWaitChange Parser
}

func (T *conn) setParse(p Parser) {
	T.parserWaitChange = p
}

func (T *conn) parserChange() {
	if T.parserWaitChange == nil {
		if _, ok := T.parser.(*defaultParse); !ok {
			T.parser = new(defaultParse)
		}
		return
	}
	// 警告：
	// 因为上面他需要初始创建一个解析接口
	// 不要移动到上面，就在这里。
	if T.parserWaitChange == T.parser {
		return
	}
	T.parser = T.parserWaitChange
}

func (T *conn) rawControl(f func(net.Conn, *bufio.Reader) error) error {
	if T.closed {
		return ErrConnClose
	}

	if T.hijackedv.isTrue() {
		return ErrHijacked
	}

	if T.handleFunc != nil {
		return ErrRwaControl
	}

	T.handleFunc = func(c net.Conn, r *bufio.Reader) error {
		T.mu.Lock()
		defer T.mu.Unlock()
		T.handleFunc = nil
		return f(c, r)
	}
	return nil
}

func (T *conn) callRawControl() bool {
	if hf := T.handleFunc; hf != nil {
		if err := hf(T.vc, T.bufr); err != nil {
			T.server.logf(LogErr, "viot: 从IP(%v)处理原始数据错误:%v", T.remoteAddr, err)
			return false
		}
		if err := T.idleWait(); err != nil {
			// 等待数据，读取超时就退出
			return false
		}
	}
	return true
}

// 劫持连接
func (T *conn) hijackLocked() (conn net.Conn, buf *bufio.ReadWriter, err error) {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.closed {
		return nil, nil, ErrConnClose
	}

	// 判断是否有主动请求
	if T.inLaunch() {
		return nil, nil, ErrLaunched
	}

	// 处理原始数据，防止冲突
	if T.handleFunc != nil {
		return nil, nil, ErrRwaControl
	}

	// 判断劫持
	if T.hijackedv.setTrue() {
		return nil, nil, ErrHijacked
	}

	// 支持后台读取，判断连接断开通知
	T.vc.DisableBackgroundRead(true)

	T.setState(StateHijacked)

	// 回收缓冲对象，由于创建使用的缓冲比较大
	putBufioWriter(T.bufw)
	T.bufw = nil

	return T.vc, bufio.NewReadWriter(T.bufr, bufio.NewWriter(T.vc)), nil
}

func (T *conn) inLaunch() bool {
	return len(T.activeReq) != 0
}

// 发射，同一时间仅接收一台客户端与设备连接，其它上锁等待
func (T *conn) RoundTrip(req *Request) (resp *Response, err error) {
	return T.RoundTripContext(req.Context(), req)
}

func (T *conn) RoundTripContext(ctx context.Context, req *Request) (resp *Response, err error) {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.closed {
		return nil, ErrConnClose
	}

	if T.hijackedv.isTrue() {
		return nil, ErrHijacked
	}

	// 处理原始数据，请求将得不出回应。
	if T.handleFunc != nil {
		return nil, ErrRwaControl
	}

	// 请求不能为空
	if req == nil {
		return nil, ErrReqUnavailable
	}

	if T.activeReq == nil {
		T.activeReq = make(map[string]chan *Response)
	}

	// 设备支持的请求格式转字节
	reqByte, err := T.parser.Unrequest(req)
	if err != nil {
		return nil, err
	}

	done := make(chan *Response)
	T.activeReq[req.nonce] = done
	defer close(done)
	defer delete(T.activeReq, req.nonce)

	if err = T.writeLineByte(reqByte); err != nil {
		return nil, err
	}

	T.mu.Unlock()
	defer T.mu.Lock()

	// 如果ctx没有设置超时，同时设备没有返回响应。结果将会阻塞，造成死锁。
	// 在此为上下文加上服务器的读取超时等待响应
	var (
		rCtx   = req.Context()
		cancel func()
	)
	if _, ok := ctx.Deadline(); !ok {
		if _, ok := rCtx.Deadline(); !ok {
			if d := T.server.ReadTimeout; d != 0 {
				ctx, cancel = context.WithTimeout(ctx, d)
				defer cancel()
			}
		}
	}

	select {
	case <-ctx.Done():
		// 上下文取消
		return nil, ctx.Err()
	case <-rCtx.Done():
		// 客户端请求结束
		return nil, rCtx.Err()
	case <-T.ctx.Done():
		// 当前连接关闭
		return nil, ErrConnClose
	case res := <-done:
		// 设备返回一个响应
		res.Request = req
		return res, nil
	}
}

func (T *conn) roundTripReceive(resp *Response) {
	T.mu.Lock()
	defer T.mu.Unlock()
	T.setState(StateActive)
	if cres, ok := T.activeReq[resp.nonce]; ok {
		select {
		case cres <- resp:
		default:
		}
	}
}

// 写入一行数据
func (T *conn) writeLineByte(b []byte) error {
	// 设置写入超时
	if d := T.server.WriteTimeout; d != 0 {
		T.rwc.SetWriteDeadline(time.Now().Add(d))
	}

	T.logDebugWriteData(bytes.TrimSuffix(b, []byte{0x0a}))

	// 客户发送一个请求到设备
	n, err := T.bufw.Write(b)
	if err != nil {
		return err
	}
	if rbn := len(b); n != rbn {
		return fmt.Errorf("actual data length %d, Length of sent data %d", rbn, n)
	}
	T.bufw.Flush()
	return nil
}

// 读取一行数据
func (T *conn) readLineBytes() (b []byte, err error) {
	if T.hijackedv.isTrue() {
		return nil, ErrHijacked
	}

	if d := T.server.ReadTimeout; d != 0 {
		T.rwc.SetReadDeadline(time.Now().Add(d))
	}

	// 设置读取限制大小
	// 恢复读取大小限制
	T.vc.SetReadLimit(T.server.maxLineBytes())
	defer T.vc.SetReadLimit(0)

	// 读取行格式
	b, err = readLineBytes(T.bufr)
	if err != nil {
		return nil, err
	}

	T.logDebugReadData(b)

	return b, err
}

// 解析响应
func (T *conn) readResponse(ctx context.Context, lineBytes []byte) (res *Response, err error) {
	if T.hijackedv.isTrue() {
		return nil, ErrHijacked
	}
	return T.parser.Response(lineBytes)
}

// 解析请求
func (T *conn) readRequest(ctx context.Context, lineBytes []byte) (req *Request, err error) {
	if T.hijackedv.isTrue() {
		return nil, ErrHijacked
	}

	req, err = T.parser.Request(lineBytes)
	if err != nil {
		return nil, err
	}

	if req.ProtoMajor != 1 {
		return nil, errors.New("unsupported protocol version")
	}

	if req.Host != "" && !httpguts.ValidHostHeader(req.Host) {
		return nil, errors.New("malformation Host")
	}

	req.ctx, req.cancelCtx = context.WithCancel(ctx)
	req.RemoteAddr = T.remoteAddr
	req.TLS = T.tlsState
	return
}

// 服务
func (T *conn) serve(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil && err != ErrAbortHandler {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			T.server.logf(LogErr, "viot: 工作意外错误 %v: %v\n%s", T.remoteAddr, err, buf)
		}
		if T.hijackedv.isTrue() {
			T.server.logf(LogDebug, "viot: 自IP(%s)使用权已交给Hijacked", T.remoteAddr)
			return
		}
		T.server.logf(LogDebug, "viot: 自IP(%s)断开网络", T.remoteAddr)
		T.close()
	}()

	T.server.logf(LogDebug, "viot: 自IP(%s)连接网络", T.remoteAddr)

	// 连接的上下文
	ctx = context.WithValue(ctx, LocalAddrContextKey, T.rwc.LocalAddr())
	T.ctx, T.cancelCtx = context.WithCancel(ctx)
	defer T.cancelCtx()

	T.remoteAddr = T.rwc.RemoteAddr().String()

	if tlsConn, ok := T.rwc.(*tls.Conn); ok {
		if d := T.server.ReadTimeout; d != 0 {
			T.rwc.SetReadDeadline(time.Now().Add(d))
		}
		if d := T.server.WriteTimeout; d != 0 {
			T.rwc.SetWriteDeadline(time.Now().Add(d))
		}
		if err := tlsConn.Handshake(); err != nil {
			T.server.logf(LogErr, "viot: TLS 握错误 %s: %v", T.remoteAddr, err)
			return
		}
		T.tlsState = new(tls.ConnectionState)
		*T.tlsState = tlsConn.ConnectionState()
		// 待验证证书请求的协议
		// NegotiatedProtocol 是客户端携带过来的
		// TLSNextProto 是服务处理该协议的
		if proto := T.tlsState.NegotiatedProtocol; validNPN(proto) {
			if fn := T.server.TLSNextProto[proto]; fn != nil {
				h := initNPNRequest{ctx, T.server, tlsConn}
				fn(T.server, tlsConn, h)
			}
			return
		}
	}

	T.vc = vconn.New(T.rwc)
	T.bufr = newBufioReader(T.vc)
	T.bufw = newBufioWriterSize(T.vc, 4<<10)

	for {
		if T.server.shuttingDown() {
			// 服务器已经下线
			return
		}

		if !T.inLaunch() {
			// 内部创建解析接口
			T.parserChange()

			// 自定义数据流处理函数
			if !T.callRawControl() {
				return
			}
		}

		lineBytes, err := T.readLineBytes()
		if err != nil {
			if isCommonNetReadError(err) {
				return
			}

			T.logErrReceive(err)
			return
		}

		// 开始发数据，前面有很多空行。需要跳过空行
		// 这样的情况需要处理\n\n\n\n\n{....}\n
		if len(lineBytes) == 0 {
			continue
		}

		// 设备发来请求，等待服务器响应信息
		req, err := T.readRequest(T.ctx, lineBytes)
		// 不是有效请求
		if err == ErrReqUnavailable && T.inLaunch() {
			res, err := T.readResponse(T.ctx, lineBytes)
			if err != nil {
				T.logErrReceive(err)
				// 不能识别的数据
				return
			}

			// 给使用者返回响应
			T.roundTripReceive(res)

			if err = T.idleWait(); err != nil {
				// 等待数据，读取超时就退出
				return
			}
			continue
		}
		if err != nil {
			etxt := fmt.Sprintf("{\"nonce\":\"-1\",\"status\":400,\"header\":{\"Connection\":\"close\"},\"body\":%q}\n", "Bad Request: "+err.Error())
			T.writeLineByte([]byte(etxt))
			T.closeWriteAndWait()
			return
		}

		T.setState(StateActive)
		w := &responseWrite{
			conn:   T,
			req:    req,
			header: make(Header),
		}
		w.dw.res = w

		// 设置写入超时时间,仅用于 w.Write 方法写入时间限制
		if d := T.server.WriteTimeout; d != 0 {
			T.rwc.SetWriteDeadline(time.Now().Add(d))
		}

		// 这里内部不能 go func 和 ctx 一起使用。否则会被取消
		serverHandler{T.server}.ServeIOT(w, req)
		req.cancelCtx()

		// 劫持
		// 连接非法关闭
		if T.hijackedv.isTrue() || T.closed {
			return
		}

		// 设置完成，生成body，发送至客户端
		if err := w.done(); err != nil {
			T.logErrSend(err)
			return
		}

		// 不能重用连接，客户端 或 服务端设置了不支持重用
		if w.closeAfterReply {
			T.closeWriteAndWait()
			return
		}

		// 不支持长连接或服务器已经下线
		if !T.server.doKeepAlives() {
			return
		}

		if err = T.idleWait(); err != nil {
			// 等待数据，读取超时就退出
			return
		}
	}
}

// 设置连接状态
func (T *conn) setState(state ConnState) {
	switch state {
	case StateNew:
		T.server.trackConn(T, true)
	case StateHijacked, StateClosed:
		T.server.trackConn(T, false)
	}
	T.curState.Store(state)
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

// 空闲等待，自动处理多余的换行符
func (T *conn) idleWait() error {
	T.setState(StateIdle)

	if d := T.server.idleTimeout(); d != 0 {
		T.rwc.SetReadDeadline(time.Now().Add(d))
	}
	for {
		c, err := T.bufr.ReadByte()
		if err != nil {
			return err
		}
		// 跳过\n\r
		if bytes.ContainsAny([]byte{c}, "\r\n") {
			continue
		}
		T.bufr.UnreadByte()
		break
	}
	if _, err := T.bufr.Peek(4); err != nil {
		return err
	}
	T.rwc.SetReadDeadline(time.Time{})
	return nil
}

// 回收缓冲对象
func (T *conn) finalFlush() {
	// 如果连接是被劫持，不支持调用此函数，否则爆panic
	if T.bufr != nil {
		putBufioReader(T.bufr)
		T.bufr = nil
	}
	if T.bufw != nil {
		T.bufw.Flush()
		putBufioWriter(T.bufw)
		T.bufw = nil
	}
}

// rstAvoidanceDelay是在关闭整个套接字之前关闭TCP连接的写入端之后我们休眠的时间量。
// 通过睡眠，我们增加了客户端看到我们的FIN并处理其最终数据的机会，然后再处理后续的RST，从而关闭已知未读数据的连接。
// 这个RST似乎主要在BSD系统上。 （和Windows？）这个超时有点武断（大概的延迟）。
const rstAvoidanceDelay = 500 * time.Millisecond

type closeWriter interface {
	CloseWrite() error
}

var _ closeWriter = (*net.TCPConn)(nil)

// 关闭并写入
func (T *conn) closeWriteAndWait() {
	if tcp, ok := T.rwc.(closeWriter); ok {
		tcp.CloseWrite()
	}
	time.Sleep(rstAvoidanceDelay)
}

// 关闭连接
func (T *conn) close() error {
	// 需要上锁，否则会清空T.bufr 或 T.bufw
	T.mu.Lock()
	defer T.mu.Unlock()

	T.closed = true

	// 取消连接上下文
	T.cancelCtx()

	// 释放缓冲对象
	defer T.finalFlush()
	T.setState(StateClosed)
	return T.rwc.Close()
}

func (T *conn) logDebugWriteData(a interface{}) {
	T.server.logf(LogDebug, "viot: 往IP(%s)写入数据:\n%s", T.remoteAddr, a)
}

func (T *conn) logDebugReadData(a interface{}) {
	T.server.logf(LogDebug, "viot: 从IP(%s)读取数据:\n%s", T.remoteAddr, a)
}

func (T *conn) logErrReceive(err error) {
	T.server.logf(LogErr, "viot: 从IP(%v)接收数据错误:%v", T.remoteAddr, err)
}

func (T *conn) logErrSend(err error) {
	T.server.logf(LogErr, "viot: 从IP(%v)发送数据错误:%v", T.remoteAddr, err)
}
