package viot

import (
	"time"
    "github.com/456vv/verror"
    "github.com/456vv/vmap/v2"
    "github.com/456vv/vweb/v2"
    "sync"
)

var DefaultHomePool    = NewHomePool()                                                      // 池（默认）


//HomePool 网站池
type HomePool struct {
	pool					sync.Map                                                        // map[host]*Home
    tick					*time.Ticker													// 定时器
    exit                    chan bool                                                       // 退出
    run						atomicBool														// 已经启动
}

func NewHomePool() *HomePool {
   	sp := &HomePool{
        exit: make(chan bool),
    }
    sp.tick = time.NewTicker(time.Second)
    return sp
}


//NewHome 创建一个家，如果存在返回已经存在的。Sessions 使用默认的设置，你需要修改它。
//默认创建的 Sessions 会话超时为1小时
//	name string		家name
//	*Home			家
func (T *HomePool) NewHome(name string) *Home {
	if inf, ok := T.pool.Load(name); ok {
		return inf.(*Home)
	}
	home := &Home{
		identity:	name,
		Sessions:	&vweb.Sessions{
			Expired: time.Hour,
		},
		Global: 	vmap.NewMap(),
	}
	T.pool.Store(name, home)
	return home
}

//DelHome 删除家
//	name string		家name
func (T *HomePool) DelHome(name string) {
	T.pool.Delete(name)
}

//RangeHome 迭举家
//	f func(name string, home *Home) bool
func (T *HomePool) RangeHome(f func(name string, home *Home) bool){
	T.pool.Range(func(key, value interface{}) bool {
		return f(key.(string), value.(*Home))
	})
}

//SetRecoverSession 设置回收无效的会话。默认为1秒
//	d time.Duration     回收时间隔，不可以等于或小于0，否则CPU爆增
func (T *HomePool) SetRecoverSession(d time.Duration) {
    T.tick.Reset(d)
}

//Start 启动池，用于读取处理过期的会话
//	error	错误
func (T *HomePool) Start() error {
    if T.run.setTrue() {
    	return verror.TrackError("viot: Home pool is enabled!")
    }
    go T.start()
	return nil
}
func (T *HomePool) start() {
    L:for {
    	select {
		case <-T.tick.C:
            T.pool.Range(func(host, inf interface{}) bool {
            	if home, ok := inf.(*Home); ok {
                	go home.Sessions.ProcessDeadAll()
            	}
                return true
            })
	    case <-T.exit:
            break L
    	}
    }
}

//Close 关闭池
//	error   错误
func (T *HomePool) Close() error {
	if !T.run.setFalse() {
    	T.exit <- true
		T.tick.Stop()
	}
    return nil
}


//Home 家数据存储
type Home struct {
    Sessions	*vweb.Sessions                                                      // 会话集
    Global		vweb.Globaler                                                       // Global
    RootDir		func(path string) string											// 网站的根目录
    Extend		interface{}															// 接口类型，可以自己存在任何类型
	identity	string
}

// PoolName 网站池名称
//	string	名称
func (T *Home) PoolName() string {
	return T.identity
}


type HomeMan struct {
	home	sync.Map
}

//SetHome 设置一个家
//	host string		家host
//	*Home			家
func (T *HomeMan) Add(host string, home *Home) {
	if home == nil {
		T.home.Delete(host)
		return
	}
	T.home.Store(host, home)
}

//GetHome 读取一个家
//	host string		家host
//	*Home			家
//	bool			true存在，否则没有
func (T *HomeMan) Get(host string) (*Home, bool){
	if inf, ok := T.home.Load(host); ok {
		return inf.(*Home), ok
	}
	
	return T.derogatoryDomain(host)
}

//Range 迭举家
//	f func(host string, home *Home) bool
func (T *HomeMan) Range(f func(host string, home *Home) bool){
	T.home.Range(func(key, value interface{}) bool {
		return f(key.(string), value.(*Home))
	})
}

//读出家，支持贬域名。
func (T *HomeMan) derogatoryDomain(host string) (s *Home, ok bool) {
    var inf interface{}
	derogatoryDomain(host, func(domain string) bool {
        inf, ok = T.home.Load(domain)
        return ok
	});
	if ok {
        return inf.(*Home), ok
    }
    return nil, false
}


