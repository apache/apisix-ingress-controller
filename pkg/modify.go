package pkg

import "fmt"

// 一键同步pod级别的target
func SyncTargetFromK8s(){
	// 对比找到不匹配的upstream && 修改upstream下的target
	//if conf.BLevel.LevelSpec != nil && conf.BLevel.LevelSpec.Pod != nil {
	//	for _, svcName := range conf.BLevel.LevelSpec.Pod{
	//		syncTargets(svcName)
	//	}
	//}
}

func SyncTargetWithUpstream(upstream string){
	syncTargets(upstream)
}

func syncTargets(svcName string){
	logger.Debug(fmt.Sprintf("开始对比 %s", svcName))
	// k8s
	pods := ListPodsBySvcName(svcName)
	// 对比
	infos := make([]podInfo, 0)
	for _, pod := range pods {
		if len(pod.Spec.Containers) > 0 && len(pod.Spec.Containers[0].Ports) > 0 {
			port := pod.Spec.Containers[0].Ports[0].ContainerPort
			infos = append(infos, podInfo{PodIp: pod.Status.PodIP, Port: port})
		}
	}
	cleanSvcMapByName(svcName)
	addSliceSvcMap(svcName, infos)
	UpdateNodes(svcName, TransPodList(svcName))
	logger.Info(fmt.Sprintf("%s 同步成功", svcName))
}

