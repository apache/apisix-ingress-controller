package pkg

import (
	"time"
	"gopkg.in/resty.v1"
	"fmt"
	"github.com/iresty/ingress-controller/conf"
	"encoding/json"
	"strings"
)

const timeout = 3000
const roundrobin = "roundrobin"
var base_url = conf.GetURL()

type UpstreamsResponse struct {
	Upstreams Upstreams `json:"node"`
}

type Upstreams struct{
	Key string `json:"key"` // 用来定位upstreams 列表
	Upstreams []Upstream `json:"nodes"`
}

type UpstreamResponse struct {
	Action string `json:"action"`
	Upstream Upstream `json:"upstream"`
}

type Upstream struct {
	Key string `json:"key"` // upstream key
	UpstreamNodes UpstreamNodes `json:"value"`
}

type UpstreamNodes struct {
	Nodes map[string]int64 `json:"nodes"`
	Desc string `json:"desc"` // upstream name  = k8s svc
	LBType string `json:"type"` // 负载均衡类型
}

//{"type":"roundrobin","nodes":{"10.244.10.11:8080":100},"desc":"somesvc"}
type UpstreamRequest struct {
	LBType string `json:"type"`
	Nodes map[string]int64 `json:"nodes"`
	Desc string `json:"desc"`
}

func FindUpstreamByName(name string) Upstream{
	url := fmt.Sprintf("%s/upstreams", base_url)
	logger.Info(fmt.Sprintf("===================>%s", url))
	ret, _ := get(url)
	var upstreamsResponse UpstreamsResponse
	if err := json.Unmarshal(ret, &upstreamsResponse); err != nil {
		logger.Error(err.Error())
	} else {
		for _, u := range upstreamsResponse.Upstreams.Upstreams {
			//fmt.Println(u.UpstreamNodes)
			if u.UpstreamNodes.Desc == name {
				return u
			}
		}
	}
	return Upstream{}
}

func AddUpstream(name string, podMap map[string]int64, lb string) (*UpstreamResponse, error){
	url := fmt.Sprintf("%s/upstreams", base_url)
	if lb == ""{
		lb = roundrobin
	}
	ur := &UpstreamRequest{LBType: lb, Nodes: podMap, Desc: name}
	if b, err := json.Marshal(ur); err != nil {
		logger.Error(err.Error())
		return nil, err
	}else {
		if bytes, err := post(url, b); err != nil {
			logger.Error(fmt.Sprintf(conf.AddUpstreamError.Msg, name))
			return nil, err
		} else {
			var uRes UpstreamResponse
			if err = json.Unmarshal(bytes, &uRes); err != nil {
				logger.Error(fmt.Sprintf(conf.AddUpstreamJsonError.Msg, name))
				return nil, err
			}else {
				return &uRes, nil
			}
		}
	}

}


func UpdateNodes(upstreamName string, podMap map[string]int64) (bool, error){
	// 根据upstream名称匹配，找到upstream id
	upstream := FindUpstreamByName(upstreamName)
	logger.Info(upstreamName)
	logger.Info(upstream)
	// 根据upstream名称匹配 podList
	if upstream.Key != ""{ // patch upstream
		// 更新upstream - id map
		UpdateEtcdUpstream(upstream.UpstreamNodes.Desc, upstream.Key)
		// patch
		url := fmt.Sprintf("%s%s/nodes", base_url, ReplacePrefix(upstream.Key))
		logger.Info(fmt.Sprintf("===================>%s", url))
		if _, err := patch(url, podMap); err != nil {
			logger.Error(fmt.Sprintf(conf.UpdateUpstreamNodesError.Msg, upstream.UpstreamNodes.Desc))
			return false, err
		}
	} else { // add upstream
		if uRes, err := AddUpstream(upstreamName, podMap, ""); err != nil {
			return false, err
		} else {
			// 更新upstream - id map
			UpdateEtcdUpstream(uRes.Upstream.UpstreamNodes.Desc, uRes.Upstream.Key)
		}
	}
	return true, nil
}

func ReplacePrefix(key string) string {
	return strings.Replace(key, "/apisix", "", 1)
}




func get(url string) ([]byte, error){
	r := resty.New().
		SetTimeout(time.Duration(timeout)*time.Millisecond).
		R().
		SetHeader("content-type", "application/json")
	resp, err := r.Get(url)
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}

func getOrigin(url string) (*resty.Response, error){
	r := resty.New().
		SetTimeout(time.Duration(timeout)*time.Millisecond).
		R().
		SetHeader("content-type", "application/json")
	resp, err := r.Get(url)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func post(url string, bytes []byte) ([]byte, error){
	r := resty.New().
		SetTimeout(time.Duration(timeout)*time.Millisecond).
		R().
		SetHeader("content-type", "application/json")
	r.SetBody(bytes)
	resp, err := r.Post(url)
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}

func patch(url string, tr map[string]int64) ([]byte, error){
	r := resty.New().
		SetTimeout(time.Duration(timeout)*time.Millisecond).
		R().
		SetHeader("content-type", "application/json")
	b, err := json.Marshal(tr)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	r.SetBody(b)
	resp, err := r.Patch(url)
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}


