package viot


import (
	"testing"
    "time"
)




func Test_Sessions_processDeadAll(t *testing.T){
    nss := Sessions{}
    nss.Expired = time.Second

    ns := &Session{}
    err := ns.Defer(t.Log, "1", "2", []string{}, "看到这里，表示Session.Defer 成功执行")
    if err != nil {
    	t.Fatal(err)
    }
    nss.SetSession("A", ns)
    time.Sleep(time.Second*2)
    
    ns1 := &Session{}
    nss.SetSession("B", ns1)
    nss.ProcessDeadAll()
    
    if nss.ss.Has("A") {
    	t.Fatal("无法删除过期Session条目")
    }
    if !nss.ss.Has("B") {
    	t.Fatal("误删除未过期Session条目")
    }
    time.Sleep(time.Second)
}


func Test_Sessions_triggerDeadSession(t *testing.T){

    nss := Sessions{}
    nss.Expired=time.Second

    ns := &Session{}
    err := ns.Defer(t.Log, "1", "2", []string{}, "看到这里，表示Session.Defer 成功执行")
    if err != nil {
    	t.Fatal(err)
    }
    nss.SetSession("A", ns)
	mse := nss.ss.Get("A").(*manageSession)
    ok := nss.triggerDeadSession(mse)
    if ok {
    	t.Fatal("错误的手工判断会话已经过期。")
    }

    time.Sleep(time.Second*1)

    ok = nss.triggerDeadSession(mse)
    if !ok {
    	t.Fatal("无法手工判断会话是否已经过期。")
    }
    time.Sleep(time.Second)

}





