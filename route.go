package viot

import(
	"sync"
	"regexp"
	"fmt"
)

type Route struct{
	HandlerError	func(w ResponseWriter, r *Request)
	rt       		sync.Map		// 路由表 map[string]
}


//HandleFunc 绑定处理函数，匹配的网址支持正则，这说明你要严格的检查。
//	url string                                          网址，支持正则匹配
//	handler func(w ResponseWriter, r *Request)    		处理函数
func (T *Route) HandleFunc(url string,  handler func(w ResponseWriter, r *Request)){
	if handler == nil {
    	T.rt.Delete(url)
		return
	}
    T.rt.Store(url, HandlerFunc(handler))
}

//ServeIOT 服务IOT
//	w ResponseWriter    响应
//	r *Request          请求
func (T *Route) ServeIOT(w ResponseWriter, r *Request){
	path := r.URL.Path
	inf, ok := T.rt.Load(r.URL.Path)
	if ok {
		inf.(Handler).ServeIOT(w, r)
		if path == r.URL.Path {
			return
		}
	}else{
		var handleFunc Handler
		T.rt.Range(func(k, v interface{}) bool {
	        regexpRegexp, err := regexp.Compile(k.(string))
	        if err != nil {
	            return true
	        }
	        _, complete := regexpRegexp.LiteralPrefix()
	        if !complete {
           		regexpRegexp.Longest()
		        if regexpRegexp.MatchString(r.URL.Path) {
		        	ok = true
		            handleFunc = v.(Handler)
		            return false
		        }
	        }
			return true
		});
		if ok {
			handleFunc.ServeIOT(w, r)
			if path == r.URL.Path {
			return
		}
		}
	}
	
	//处理错误的请求
	if T.HandlerError != nil {
		T.HandlerError(w, r)
		return
	}
	
	//默认的错误处理
	w.Status(404)
	w.Header().Set("Connection","close")
	w.SetBody(fmt.Sprintf("The path does not exist %s", path))
}