package viot
import(
	"net"
    "github.com/456vv/vmap/v2"
    "context"
    "bufio"
)

type DotContexter interface{
    Context() context.Context                                             					// 上下文
    WithContext(ctx context.Context)														// 替换上下文
}
// TemplateDoter 可以在模本中使用的方法
type TemplateDoter interface{
    RootDir(path string) string																// 家的根目录
    Request() *Request                                                                 		// 用户的请求信息
    Header() Header                                                                    		// 标头
    ResponseWriter() ResponseWriter                                                    		// 数据写入响应
    Launch() RoundTripper																	// 发射
    Hijack() (net.Conn, *bufio.ReadWriter, error)											// 劫持
    Session(token string) Sessioner                                                         // 用户的会话缓存
    Global() Globaler                                                                       // 全站缓存
    Swap() *vmap.Map                                                                        // 信息交换
    Defer(call interface{}, args ... interface{}) error										// 退回调用
    DotContexter																			// 上下文
}


//模板点
type TemplateDot struct {
    R     				*Request                                                               		// 请求
    W    				ResponseWriter                                                         		// 响应
    Home       		 	*Home                                                                       // 家配置
    Writed      		bool                                                                        // 表示已经调用写入到客户端。这个是只读的
    exchange       		vmap.Map                                                                    // 缓存映射
    ec					exitCall																	// 退回调用函数
    ctx					context.Context																// 上下文
}

//RootDir 家的根目录
//	upath string	页面路径
//	string 			根目录
func (T *TemplateDot) RootDir(upath string) string {
	if T.Home != nil && T.Home.RootDir != nil {
		return T.Home.RootDir(upath)
	}
	return "."
}

//Request 用户的请求信息
//	*Request 请求
func (T *TemplateDot) Request() *Request {
    return T.R
}

//Header 标头
//	Header   响应标头
func (T *TemplateDot) Header() Header {
    return T.W.Header()
}

//ResponseWriter 数据写入响应，调用这个接口后，模板中的内容就不会显示页客户端去
//	ResponseWriter      响应
func (T *TemplateDot) ResponseWriter() ResponseWriter {
	T.Writed=true
    return T.W
}

//Launch 发射数据到设备
//	RoundTripper  转发
func (T *TemplateDot) Launch() RoundTripper {
    return T.W.(Launcher).Launch()
}

//劫持连接
//	net.Conn			原连接
//	*bufio.ReadWriter	读取缓冲
//	error				错误
func (T *TemplateDot) Hijack() (net.Conn, *bufio.ReadWriter, error) {
    return T.W.(Hijacker).Hijack()
}

//Session 用户的会话缓存
//	token string	令牌
//	Sessioner  		会话缓存
func (T *TemplateDot) Session(token string) Sessioner {	
	if T.Home == nil || T.Home.Sessions == nil {
		return nil
	}
	session, ok := T.Home.Sessions.GetSession(token)
	if !ok {
		return T.Home.Sessions.SetSession(token, &Session{})
	}
	return session
}

//Global 全站缓存
//	Globaler	公共缓存
func (T *TemplateDot) Global() Globaler {
	if T.Home == nil || T.Home.Global == nil {
		return nil
	}
    return T.Home.Global
}

//Swap 信息交换
//	Swaper  映射
func (T *TemplateDot) Swap() *vmap.Map {
    return &T.exchange
}

//Defer 在用户会话时间过期后，将被调用。
//	call interface{}            函数
//	args ... interface{}        参数或更多个函数是函数的参数
//	error                       错误
//  例：
//	.Defer(fmt.Println, "1", "2")
//	.Defer(fmt.Printf, "%s", "汉字")
func (T *TemplateDot) Defer(call interface{}, args ... interface{}) error {
    return T.ec.Defer(call, args...)
}

//Free 释放Defer
func (T *TemplateDot) Free() {
    T.ec.Free()
}

//Context 上下文
//	context.Context 上下文
func (T *TemplateDot) Context() context.Context {
	if T.ctx != nil {
		return T.ctx
	}
	return context.Background()
}

//WithContext 替换上下文
//	ctx context.Context 上下文
func (T *TemplateDot) WithContext(ctx context.Context) {
	if ctx == nil {
		panic("nil context")
	}
	T.ctx = ctx
}
