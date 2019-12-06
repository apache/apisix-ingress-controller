package pkg

import (
	"k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"github.com/iresty/ingress-controller/conf"
	"fmt"
	"net/http"
	"github.com/iresty/ingress-controller/log"
)


var logger = log.GetLogger()
var errorCallbacks = make([]func(error), 0)

const ADD = "ADD"
const UPDATE = "UPDATE"
const DELETE = "DELETE"

type podInfo struct {
	PodIp string `json:"podIp"`
	Port int32 `json:"port"`
}
var podMap = make(map[string]*podInfo) // key: podName
var svcMap = make(map[string][]podInfo) // key: svcName

type controller struct {
	queue chan interface{}
}

type queueObj struct {
	OpeType string `json:"ope_type"`
	Obj interface{} `json:"obj"`
}

func (c *controller) pop() interface{}{
	e := <- c.queue
	return e
}

func (c *controller) run() {
	for {
		//c.pop()
		ele := c.pop()
		c.process(ele)
	}
}

func distinctPodInfo(arr []podInfo) (newArr []podInfo) {
	newArr = make([]podInfo, 0)
	for i := 0; i < len(arr); i++ {
		a := fmt.Sprintf("%s:%d", arr[i].PodIp, arr[i].Port)
		repeat := false
		for j := i + 1; j < len(arr); j++ {
			b := fmt.Sprintf("%s:%d", arr[j].PodIp, arr[j].Port)
			if a == b {
				repeat = true
				break
			}
		}
		if !repeat {
			newArr = append(newArr, arr[i])
		}
	}
	return newArr
}

func removeFromPodInfos(s podInfo, f []podInfo) []podInfo{
	a := fmt.Sprintf("%s:%d", s.PodIp, s.Port)
	for k, v := range f {
		b := fmt.Sprintf("%s:%d", v.PodIp, v.Port)
		if a == b {
			return append(f[:k], f[k+1:]...)
		}
	}
	return f
}

func addSvcMap(name string, info *podInfo){
	infos := svcMap[name]
	if infos == nil {
		infos = make([]podInfo, 0)
	}
	svcMap[name] = distinctPodInfo(append(infos, *info))
}

func addSliceSvcMap(name string, info []podInfo){
	infos := svcMap[name]
	if infos == nil {
		infos = make([]podInfo, 0)
	}
	svcMap[name] = distinctPodInfo(append(infos, info...))
}

func removeSvcMap(name string, info *podInfo){
	infos := svcMap[name]
	if infos != nil  {
		svcMap[name] = removeFromPodInfos(*info, infos)
	}
}

func cleanSvcMapByName(name string) {
	svcMap[name] = nil
}

func TransPodList(name string) map[string]int64 {
	infos := svcMap[name]
	result := make(map[string]int64)
	for _, p := range infos {
		s := fmt.Sprintf("%s:%d", p.PodIp, p.Port)
		result[s] = 100 //权重默认100
	}
	return result
}

func (c *controller) process(obj interface{}) {
	qo, _ := obj.(*queueObj)
	pod, _ := qo.Obj.(*v1.Pod)
	svcName := pod.Annotations["app_name"]
	name := pod.Name
	podIp := pod.Status.PodIP
	logger.Info(svcName, podIp, qo.OpeType)
	if len(pod.Spec.Containers) > 0 && len(pod.Spec.Containers[0].Ports) > 0 {
		port := pod.Spec.Containers[0].Ports[0].ContainerPort
		if port != 0 && podIp != "" {
			// 记录下pod的ip和端口号
			podMap[name] = &podInfo{PodIp:podIp, Port:port}
		}
	}
	podInfo := podMap[name]
	if svcName != "" && podInfo != nil {
		if qo.OpeType == DELETE {
			// svcMap 删除 podInfo
			removeSvcMap(svcName, podInfo)
			// 整体更新
			if conf.IsLeader { //是leader 才更新
				if _, err := UpdateNodes(svcName, TransPodList(svcName)); err != nil {
					logger.Error(err.Error())
				} else {
					delete(podMap, name)
				}
			}
		}else {
			for _, condition := range pod.Status.Conditions {
				if condition.Type == v1.PodReady && condition.Status == v1.ConditionTrue && pod.Status.Phase == v1.PodRunning { // pod Ready
					// svcMap 增加 podInfo
					addSvcMap(svcName, podInfo)
					// 整体更新
					if conf.IsLeader { //是leader 才更新
						if _, err := UpdateNodes(svcName, TransPodList(svcName)); err != nil {
							logger.Error(err.Error())
						}
					}
				} else if condition.Type == v1.PodReady { // pod not Ready
					// svcMap 删除 podInfo
					removeSvcMap(svcName, podInfo)
					// 整体更新
					if conf.IsLeader { //是leader 才更新
						if _, err := UpdateNodes(svcName, TransPodList(svcName)); err != nil {
							logger.Error(err.Error())
						} else {
							delete(podMap, name)
						}
					}
				}
			}
		}
	} else {
		logger.Error(svcName, podIp, qo.OpeType, "not process")
	}
}


type QueueEventHandler struct {
	c *controller
}

func (h *QueueEventHandler) OnAdd(obj interface{}) {
	h.c.queue <- &queueObj{ADD, obj}
}

func (h *QueueEventHandler) OnDelete(obj interface{}) {
	h.c.queue <- &queueObj{DELETE, obj}
}

func (h *QueueEventHandler) OnUpdate(old, update interface{}) {
	h.c.queue <- &queueObj{ UPDATE, update}
}

func Watch(){
	// 增加错误监控
	errorCallbacks = append(errorCallbacks, ErrorCallback)
	setupErrorHandlers()
	// informer
	stopCh := make(chan struct{})
	podInformer := conf.GetPodInformer()
	c := &controller{
		queue: make(chan interface{}, 100),
	}
	podInformer.Informer().AddEventHandler(&QueueEventHandler{c:c})
	// podInformer
	go podInformer.Informer().Run(stopCh)
	go c.run()
	// svcInformer
	svcInformer := conf.GetSvcInformer()
	go svcInformer.Informer().Run(stopCh)
	// nsInformer
	nsInformer := conf.GetNsInformer()
	go nsInformer.Informer().Run(stopCh)

	// scheduler
	go Scheduler()
	// web
	router := Route()
	err := http.ListenAndServe(":8080", router)
	if err != nil {
		logger.Fatal("ListenAndServe: ", err)
	}
}
func setupErrorHandlers() {
	nErrFunc := len(utilruntime.ErrorHandlers)
	customErrorHandler := make([]func(error), nErrFunc+1)
	copy(customErrorHandler, utilruntime.ErrorHandlers)
	customErrorHandler[nErrFunc] = func(err error) {
		for _, callback := range errorCallbacks {
			callback(err)
		}
	}
	utilruntime.ErrorHandlers = customErrorHandler
}

func ErrorCallback(err error){
	logger.Error("ALARM FROM K8S", err.Error())
}
