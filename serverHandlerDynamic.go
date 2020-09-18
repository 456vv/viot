package viot
import (
    "path/filepath"
    "io/ioutil"
    "bytes"
    "fmt"
    "bufio"
    "time"
    "os"
    "errors"
    "context"
    "runtime"
    "github.com/456vv/verror"
    "io"
)


type DynamicTemplater interface{
    SetPath(rootPath, pagePath string)																				// 设置路径
    Parse(r *bufio.Reader) (err error)																				// 解析
    Execute(out *bytes.Buffer, dot interface{}) error																// 执行
}
type DynamicTemplateFunc func() DynamicTemplater

//ServerHandlerDynamic 处理动态页面文件
type ServerHandlerDynamic struct {
	//必须的
	RootPath			string																// 根目录
    PagePath  			string																// 主模板文件路径

    //可选的
    Home        		*Home																// 网站配置
	Context				context.Context														// 上下文
	Plus				map[string]DynamicTemplateFunc										// 支持更动态文件类型
   	exec				DynamicTemplater
   	modeTime			time.Time
}

//ServeHTTP 服务HTTP
//	rw ResponseWriter    响应
//	req *Request         请求
func (T *ServerHandlerDynamic) ServeHTTP(rw ResponseWriter, req *Request){

	if T.PagePath == "" {
		T.PagePath = req.URL.Path
	}
	var filePath = filepath.Join(T.RootPath, T.PagePath)

	osFile, err := os.Open(filePath)
	if err != nil {
	    Error(rw, fmt.Sprintf("Failed to read the file! Error: %s", err.Error()), 500)
	    return
	}
	defer osFile.Close()

	//记录文件修改时间，用于缓存文件
	osFileInfo, err := osFile.Stat()
	if err != nil {
		T.exec = nil
	}else{
		modeTime := osFileInfo.ModTime()
		if !modeTime.Equal(T.modeTime) {
			T.exec = nil
		}
		T.modeTime = modeTime
	}
	
	if T.exec == nil {
	    var content, err = ioutil.ReadAll(osFile)
	    if err != nil {
	    	Error(rw, fmt.Sprintf("Failed to read the file! Error: %s", err.Error()), 500)
	        return
	    }

	    //解析模板内容
		err = T.Parse(bytes.NewBuffer(content))
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
				Error(rw, err.Error(), 500)
				return
			}
			//原样写入
			io.WriteString(rw.(io.Writer), err.Error())
			fmt.Println(err.Error())
			return
		}
		if !dock.Writed {
			//字符 设置到 主体
			rw.SetBody(body.String())
			body.Reset()
		}
	}()

	//执行模板内容
	err = T.Execute(body, (TemplateDoter)(dock))
}

//ParseText 解析模板
//	content, name string	模板内容，模板名称
//	error					错误
func (T *ServerHandlerDynamic) ParseText(content, name string) error {
	T.PagePath = name
	r := bytes.NewBufferString(content)
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
//	bufr *bytes.Reader	模板内容
//	error				错误
func (T *ServerHandlerDynamic) Parse(bufr *bytes.Buffer) (err error) {
	if T.PagePath == "" {
    	return verror.TrackError("viot: ServerHandlerDynamic.PagePath is not a valid path")
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
        var shdt = &serverHandlerDynamicTemplate{}
		shdt.SetPath(T.RootPath, T.PagePath)
        err = shdt.Parse(bufio.NewReader(bufr))
        if err != nil {
        	return
        }
        T.exec = shdt
    default:
    	if T.Plus == nil || len(dynmicType) < 3 {
    		return errors.New("viot: The file type of the first line of the file is not recognized")
    	}
		if plus, ok := T.Plus[dynmicType[2:]]; ok {
			shdt := plus()
			shdt.SetPath(T.RootPath, T.PagePath)
			err = shdt.Parse(bufio.NewReader(bufr))
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
func (T *ServerHandlerDynamic) Execute(bufw *bytes.Buffer, dock interface{}) (err error) {
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

