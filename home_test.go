package viot

import (
	"testing"
    "time"
)



func Test_Home_SetHome(t *testing.T) {
	homePool := NewHomePool()
    home := homePool.NewHome("A")
    
    homeMan := HomeMan{}
    homeMan.Add("host", home)
    if _, ok := homeMan.Get("host"); !ok {
    	t.Fatal("无法往池中增加站点")
    }
    
    //写入池
    homeMan.Add("host", nil) //删除
    if _, ok := homeMan.Get("host"); ok {
    	t.Fatal("无法从池中删除站点")
    }
	
    homeMan.Add("host", home)
    if _, ok := homeMan.Get("host"); !ok {
    	t.Fatal("无法往池中增加站点")
    }
}

func Test_Home_GetHome(t *testing.T) {
	homePool := NewHomePool()
    home := homePool.NewHome("A")
	
    homeMan := HomeMan{}
    homeMan.Add("*.viot.com:80", home)
    
    if _, ok := homeMan.Get("aaaaa.viot.com:80"); !ok {
    	t.Fatal("无法往池中增加站点")
    }
    
    //写入池
    homeMan.Add("*.viot.com:80", nil) //删除
    if _, ok := homeMan.Get("aaaaa.viot.com:80"); ok {
    	t.Fatal("无法从池中删除站点")
    }
	
    homeMan.Add("*.viot.com:80", home)
    if _, ok := homeMan.Get("bbbbbb.viot.com:80"); !ok {
    	t.Fatal("无法往池中增加站点")
    }
}


func Test_Home_Start(t *testing.T) {
	//创建池并设置刷新时间
	homePool := NewHomePool()
	homePool.Start()
    homePool.SetRecoverSession(time.Second*2)
	defer homePool.Close()
    home := homePool.NewHome("A")
    home.Sessions.Expired = time.Second
    
    //生成会话
    home.Sessions.SetSession("a", &Session{})
	
    if home.Sessions.Len() != 1 {
        t.Fatal("无法增加会话")
    }
    time.Sleep(time.Second*4)
    if home.Sessions.Len() != 0 {
        t.Fatal("无法删除过期会话")
    }
}

func Test_Home_SetRecoverSession(t *testing.T) {
	//创建池并设置刷新时间
	homePool := NewHomePool()
	homePool.Start()
	defer homePool.Close()
	
    home := homePool.NewHome("A")
    home.Sessions.Expired = time.Second*2
    
    //生成会话
    ok := false
    home.Sessions.SetSession("a", &Session{}).Defer(func(){
    	ok=true
    })
	
    if home.Sessions.Len() != 1 {
        t.Fatal("无法增加会话")
    }
    time.Sleep(time.Second*4)
    if home.Sessions.Len() != 0 {
        t.Fatal("无法删除过期会话")
    }
    if !ok {
    	t.Fatal("会话过期没有调用清除函数")
    }
}
