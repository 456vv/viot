package viot

import (
    "time"
    "github.com/456vv/vmap/v2"
)

type manageSession struct{
	s		Sessioner
	recent	time.Time
}

// Sessions集
type Sessions struct{
    Expired         time.Duration                                       // 保存session时间长
    ss        		vmap.Map                                            // 集，map[token]*Session
}

//Len 当前Session数量
//	int	数量
func (T *Sessions) Len() int {
	return T.ss.Len()
}

//ProcessDeadAll 定时来处理过期的Session
//	[]string	过期的ID名称
func (T *Sessions) ProcessDeadAll() []interface{} {
    var expId   []interface{}
	if T.Expired != 0 {
	    currTime := time.Now()
		T.ss.Range(func(token, mse interface{}) bool{
			ms := mse.(*manageSession)
	        recentTime := ms.recent.Add(T.Expired)
	        if currTime.After(recentTime) {
	        	//追加了expId一次性删除
	        	expId = append(expId, token)
	        	//执行Defer
	        	go ms.s.Free()
	        }
			return true
		})
	    T.ss.Dels(expId)
	}
    return expId
}

//triggerDeadSession 由用户来触发，并删除已挂载入的Defer
func (T *Sessions) triggerDeadSession(ms *manageSession) (ok bool) {
	if T.Expired != 0 {
	    currTime := time.Now()
	    recentTime := ms.recent.Add(T.Expired)
	     if currTime.After(recentTime) {
	        go ms.s.Free()
	        return true
	    }
	}
    return
}

//GetSession 使用token读取会话
//	token string	标识符
//	Sessioner   	会话
//	bool	       	错误
func (T *Sessions) GetSession(token string) (Sessioner, bool) {
    mse, ok := T.ss.GetHas(token)
    if !ok {
    	return nil, false
    }
    ms := mse.(*manageSession)

    if T.triggerDeadSession(ms) {
    	T.ss.Del(token)
        return nil, false
    }
    ms.recent = time.Now()
    return ms.s, true
}

//SetSession 使用token写入新的会话
//	token string   	标识符
//	s Sessioner 	新的会话
func (T *Sessions) SetSession(token string, s Sessioner) Sessioner {
	if inf, ok := T.ss.GetHas(token); ok {
		ms := inf.(*manageSession)
		if ms.s.Token() == s.Token() {
	    	//已经存在，无法再设置
	    	return s
		}
		go ms.s.Free()
	}
	if t, can := s.(*Session); can {
		//对应这个token，并保存
		t.token = token
	}
	ms := &manageSession{
		s:s,
		recent:time.Now(),
	}
    T.ss.Set(token, ms)
    return s
}

//DelSession 使用token删除的会话
//	token string   token标识符
func (T *Sessions) DelSession(token string) {
    if mse, ok := T.ss.GetHas(token); ok {
	    ms := mse.(*manageSession)
		go ms.s.Free()
		T.ss.Del(token)
    }
}
