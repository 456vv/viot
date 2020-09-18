package viot

import(
	"sync/atomic"
	"io"
)

//上下文的Key，在请求中可以使用
type contextKey struct {
	name string
}
func (T *contextKey) String() string { return "viot context value " + T.name }

//响应完成设置
type atomicBool int32
func (T *atomicBool) isTrue() bool 	{ return atomic.LoadInt32((*int32)(T)) != 0 }
func (T *atomicBool) isFalse() bool	{ return atomic.LoadInt32((*int32)(T)) != 1 }
func (T *atomicBool) setTrue() bool	{ return !atomic.CompareAndSwapInt32((*int32)(T), 0, 1)}
func (T *atomicBool) setFalse() bool{ return !atomic.CompareAndSwapInt32((*int32)(T), 1, 0)}

//空的请求解码
type noBody struct{}
func (T *noBody) Decode(i interface{}) error {return io.EOF}


// 点函数映射
var dotPackage = make(map[string]map[string]interface{})




