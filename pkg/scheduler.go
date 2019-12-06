package pkg

import (
	"github.com/iresty/ingress-controller/conf"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/api/core/v1"
)

func ListPods(m map[string]string) ([]*v1.Pod, error){
	podInformer := conf.GetPodInformer()
	selector := labels.Set(m).AsSelector()
	ret, err := podInformer.Lister().List(selector)
	for _, pod := range ret {
		logger.Info(pod.Status.PodIP)
	}
	return ret, err
}

func ListPodsBySvcName(name string) []*v1.Pod {
	svcInformer := conf.GetSvcInformer()
	selector := labels.NewSelector()
	ret, _ := svcInformer.Lister().List(selector)
	for _, svc := range ret {
		if svc.Name == name {
			logger.Debug(svc.Spec.Selector)
			if pods, err := ListPods(svc.Spec.Selector); err != nil {
				return []*v1.Pod{}
			} else {
				return pods
			}
		}
	}
	return []*v1.Pod{}
}

func Scheduler(){
	//jobrunner.Start()
	// 定时10s检测一次
	//jobrunner.Schedule("*/10 * * * * *", Compared{})
}

//type Compared struct{}

//func (w Compared) Run() {
//	CompareAndAlarm()
//}

