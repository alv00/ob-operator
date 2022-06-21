package converter

import (
	"fmt"
	cloudv1 "github.com/oceanbase/ob-operator/apis/cloud/v1"
	myconfig "github.com/oceanbase/ob-operator/pkg/config"
	observerconst "github.com/oceanbase/ob-operator/pkg/controllers/observer/const"
	"io/ioutil"
	"k8s.io/klog/v2"
	"net/http"
)

func IsAllOBAgentActive(statefulApp cloudv1.StatefulApp, obClusters []cloudv1.Cluster) bool {
	obAgentCurrentReplicas := make(map[string]bool)
	subsets := statefulApp.Status.Subsets
	podIps := subsets[0].Pods[0].PodIP
	for podIp := range podIps {
		if CheckObAgentActive(podIp) {
			//to do
			obAgentCurrentReplicas["zone1"] = true
		}
	}
	for _, obCluster := range obClusters {
		if obCluster.Cluster == myconfig.ClusterName {
			for _, zone := range obCluster.Zone {
				tmp := obAgentCurrentReplicas[zone.Name]
				if !tmp {
					return false
				}
			}
			return true
		}
	}
	return false
}

func CheckObAgentActive(podIp int) bool {
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
