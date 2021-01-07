package viot
import (
    "path/filepath"
    "io/ioutil"
    "net/url"
    "bytes"
    "strings"
    "fmt"
    "time"
    "os"
    "errors"
    "context"
    "runtime"
    "github.com/456vv/verror"
    "io"
    "log"
)


type DynamicTemplater interface{
    SetPath(rootPath, pagePath string)																			// 设置路径
    Parse(r io.Reader) (err error)																				// 解析
    Execute(out io.Writer, dot interface{}) error																// 执行
}
type DynamicTemplateFunc func(*ServerHandlerDynamic) DynamicTemplater

//ServerHandlerDynamic 处理动态页面文件
type ServerHandlerDynamic struct {
	//必须的
	RootPath			string																// 根目录
    PagePath  			string																// 主模板文件路径

    //可选的
    Home        		*Home																// 网站配置
	Context				context.Context														// 上下文
	Plus				map[string]DynamicTemplateFunc										// 支持更动态文件类型
	ReadFile			func(u *url.URL, filePath string) (io.Reader, time.Time, error)		// 读取文件。仅在 .ServeHTTP 方法中使用
	ReplaceParse		func(name string, p []byte) []byte									// 替换解析
   	exec				DynamicTemplater
   	modeTime			time.Time
}

//ServeIOT 服务IOT
//	rw ResponseWriter    响应
//	req *Request         请求
func (T *ServerHandlerDynamic) ServeIOT(rw ResponseWriter, req *Request){

	if T.PagePath == "" {
		T.PagePath = req.URL.Path
	}
	var filePath = filepath.Join(T.RootPath, T.PagePath)

	var (
		tmplread io.Reader
		modeTime time.Time
		err error
	 )
	if T.ReadFile != nil {
		tmplread, modeTime, err = T.ReadFile(req.URL, filePath)
		if err != nil {
		    Error(rw, err.Error(), 500)
		    return
		}
		if !modeTime.Equal(T.modeTime) {
			T.exec = nil
		}
		T.modeTime = modeTime
	}else{
		osFile, err := os.Open(filePath)
		if err != nil {
			err = fmt.Errorf("Failed to read the file! Error: %s", err.Error())
		    Error(rw, err.Error(), 500)
		    return
		}
		defer osFile.Close()
		tmplread = osFile

		//记录文件修改时间，用于缓存文件
		osFileInfo, err := osFile.Stat()
		if err != nil {
			T.exec = nil
		}else{
			modeTime = osFileInfo.ModTime()
			if !modeTime.Equal(T.modeTime) {
				T.exec = nil
			}
			T.modeTime = modeTime
		}
	}
	if T.exec == nil {
	    //解析模板内容
		err = T.Parse(tmplread)
	    if err != nil {
			Error(rw, err.Error(), 500)
	        return
	    }
	}

    //模板点
    var dock = &TemplateDot{
        R    	 	: req,
        W    		: rw,
        Home        : T.Home,
    }

    ctx := T.Context
    if ctx == nil {
    	ctx = req.Context()
    }
    dock.WithContext(context.WithValue(ctx, "Dynamic", T))
	var body = new(bytes.Buffer)
	defer func(){
		dock.Free()
		if err != nil {
			if !dock.Writed {
				//页面代码错误
				Error(rw, err.Error(), 500)
				return
			}
			
			io.WriteString(rw.(io.Writer), err.Error())
			log.Println(err.Error())
			return
		}
		
		if !dock.Writed {
			if body.Len() != 0 {
				rw.SetBody(body.String())
			}
		}
	}()

	//执行模板内容
	err = T.Execute(body, (TemplateDoter)(dock))
}

//ParseText 解析模板
//	content, name string	模板内容，模板名称
//	error					错误
func (T *ServerHandlerDynamic) ParseText(name, content string) error {
	T.PagePath = name
	r := strings.NewReader(content)
	return T.Parse(r)
}

//ParseFile 解析模板
//	path string			模板文件路径，如果为空，默认使用RootPath,PagePath字段
//	error				错误
func (T *ServerHandlerDynamic) ParseFile(path string) error {
	if path == "" {
		path = filepath.Join(T.RootPath, T.PagePath)
	}else if !filepath.IsAbs(path) {
		T.PagePath = path
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	defer file.Close()
	b, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	r := bytes.NewBuffer(b)
	return T.Parse(r)
}

//Parse 解析模板
//	r io.Reader			模板内容
//	error				错误
func (T *ServerHandlerDynamic) Parse(r io.Reader) (err error) {
	if T.PagePath == "" {
    	return verror.TrackError("viot: ServerHandlerDynamic.PagePath is not a valid path")
	}
	
	
	bufr, ok := r.(*bytes.Buffer)
	if T.ReplaceParse != nil {
		allb, err := ioutil.ReadAll(r)
		if err != nil {
			return  verror.TrackErrorf("viot: ServerHandlerDynamic.ReplaceParse failed to read data: %s", err.Error())
		}
		allb = T.ReplaceParse(T.PagePath, allb)
		bufr = bytes.NewBuffer(allb)
	}else if !ok {
		bufr = bytes.NewBuffer(nil)
		bufr.ReadFrom(r)
	}
	
    //文件首行
    firstLine, err := bufr.ReadBytes('\n')
    if err != nil || len(firstLine) == 0 {
    	return verror.TrackErrorf("viot: Dynamic content is empty! Error: %s", err.Error())
    }
    drop := 0
	if firstLine[len(firstLine)-1] == '\n' {
		drop = 1
		if len(firstLine) > 1 && firstLine[len(firstLine)-2] == '\r' {
			drop = 2
		}
		firstLine = firstLine[:len(firstLine)-drop]
	}

	dynmicType := string(firstLine)
    switch dynmicType {
    case "//template":
        var shdt = &serverHandlerDynamicTemplate{P:T}
		shdt.SetPath(T.RootPath, T.PagePath)
        err = shdt.Parse(bufr)
        if err != nil {
        	return
        }
        T.exec = shdt
    default:
    	if T.Plus == nil || len(dynmicType) < 3 {
    		return errors.New("viot: The file type of the first line of the file is not recognized")
    	}
		if plus, ok := T.Plus[dynmicType[2:]]; ok {
			shdt := plus(T)
			shdt.SetPath(T.RootPath, T.PagePath)
			err = shdt.Parse(bufr)
	        if err != nil {
	        	return
	        }
	       T.exec = shdt
		}
    }
    return
}

//Execute 执行模板
//	bufw *bytes.Buffer	模板返回数据
//	dock interface{}	与模板对接接口
//	error				错误
func (T *ServerHandlerDynamic) Execute(bufw io.Writer, dock interface{}) (err error) {
	if T.exec == nil {
		return errors.New("viot: Parse the template content first and then call the Execute")
	}
	defer func (){
		if e := recover(); e != nil{
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			err = fmt.Errorf("viot: Dynamic code execute error。%v\n%s", e, buf)
		}
	}()

	return T.exec.Execute(bufw, dock)
}


