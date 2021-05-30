package viot

import (
	"time"
    "github.com/456vv/vweb/v2"
)

var DefaultSitePool    = NewSitePool()                                                      // 池（默认）


//SitePool 网站池
type SitePool struct{
	*vweb.SitePool
}

func NewSitePool() *SitePool {
    return &SitePool{vweb.NewSitePool()}
}


//NewSite 创建一个站点，如果存在返回已经存在的。Sessions 使用默认的设置，你需要修改它。
//默认创建的 Sessions 会话超时为1小时
//	name string		站点name
//	*Site			站点
func (T *SitePool) NewSite(name string) *Site {
	site := T.SitePool.NewSite(name)
	site.Sessions.Expired = time.Hour
	return site
}