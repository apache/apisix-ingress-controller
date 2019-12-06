package pkg

import (
	"github.com/iresty/ingress-controller/conf"
	"context"
	"fmt"
	"encoding/json"
	"strings"
	"github.com/coreos/etcd/client"
)


func UpdateEtcdUpstream(name, key string){
	kapi := conf.GetEtcdAPI()
	resp, err := kapi.Get(context.Background(), conf.AispeechUpstreamKey, nil)
	if err != nil {
		logger.Error(fmt.Sprintf("获取etcd: %s 失败，%s" , conf.AispeechUpstreamKey, err.Error()))
		if strings.Contains(err.Error(), "Key not found") {// 如果是找不到key，则新增
			updateUpstreamMap(resp, kapi, name, key)
		}
	}else {
		updateUpstreamMap(resp, kapi, name, key)
	}
}

func updateUpstreamMap(resp *client.Response, kapi client.KeysAPI, name, key string) {
	upstreamMap := make(map[string]string)
	if resp != nil  {
		str := resp.Node.Value
		if err := json.Unmarshal([]byte(str), &upstreamMap); err != nil {
			logger.Errorf("etcd upstream map to json error: %s", err.Error())
		}
	}
	upstreamMap[name] = key
	if bytes, err := json.Marshal(upstreamMap); err != nil {
		logger.Errorf("etcd upstream map to json string error: %s", err.Error())
	}else {
		json := string(bytes)
		kapi.Set(context.Background(), conf.AispeechUpstreamKey, json, nil)
	}
}
