package pkg

import (
	"github.com/api7/ingress-controller/pkg/config"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func ListPods(m map[string]string) ([]*v1.Pod, error){
	podInformer := config.GetPodInformer()
	selector := labels.Set(m).AsSelector()
	ret, err := podInformer.Lister().List(selector)
	for _, pod := range ret {
		logger.Info(pod.Status.PodIP)
	}
	return ret, err
}

func ListPodsBySvcName(name string) []*v1.Pod {
	svcInformer := config.GetSvcInformer()
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

