module github.com/456vv/viot/v2

go 1.16

replace github.com/456vv/vconn => ../vconn

replace github.com/456vv/vweb/v2 => ../vweb

require (
	github.com/456vv/vconn v1.0.11
	github.com/456vv/vconnpool/v2 v2.1.4 // indirect
	github.com/456vv/verror v1.1.0 // indirect
	github.com/456vv/vmap/v2 v2.3.1 // indirect
	github.com/456vv/vweb/v2 v2.0.0
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5 // indirect
)
