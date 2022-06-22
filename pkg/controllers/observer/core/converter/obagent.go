package converter

import (
	"fmt"
	cloudv1 "github.com/oceanbase/ob-operator/apis/cloud/v1"
	observerconst "github.com/oceanbase/ob-operator/pkg/controllers/observer/const"
	"io/ioutil"
	"k8s.io/klog/v2"
	"net/http"
)

func IsAllOBAgentActive(statefulApp cloudv1.StatefulApp) bool {
	readiness := true
	subsets := statefulApp.Status.Subsets
	for subsetsIdx, _ := range subsets {
		for _, pod := range subsets[subsetsIdx].Pods {
			if !CheckObAgentActive(pod.PodIP) {
				readiness = false
			}
		}
	}
	return readiness
}

func CheckObAgentActive(podIp string) bool {
	client := &http.Client{}
	checkUrl := fmt.Sprintf("http://%s:%d%s", podIp, observerconst.MonagentPort, observerconst.MonagentReadinessUrl)
	req, err := http.NewRequest("GET", checkUrl, nil)
	if err != nil {
		klog.Errorln("check obAgent active: get new request", err)
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		klog.Errorln("check obAgent active: client do request", err)
		return false
	}
	defer resp.Body.Close()
	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Errorln("check obAgent active: read response", err)
		return false
	}
	klog.Infoln("check obAgent active: response body :", bodyText)
	return true
}
