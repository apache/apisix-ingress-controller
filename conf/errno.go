package conf

import "fmt"

type Message struct {
	Code string
	Msg  string
}

var (
	//AA 01 表示ingress-controller本身的错误
	//BB 00 表示系统信息
	SystemError = Message{"010001", "system errno"}

	//BB 01表示更新失败
	UpdateUpstreamNodesError = Message{"010101", "服务%s节点更新失败"}
	AddUpstreamError         = Message{"010102", "增加upstream %s失败"}
	AddUpstreamJsonError         = Message{"010103", "upstream %s json trans error"}
)

func (m Message) ToString(params ...interface{}) string{
	params = append(params, m.Code)
	return fmt.Sprintf(m.Msg + " error_code:%s", params...)
}