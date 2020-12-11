// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package pkg

import (
	"github.com/api7/ingress-controller/conf"
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

