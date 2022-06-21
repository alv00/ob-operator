package converter

import (
	cloudv1 "github.com/oceanbase/ob-operator/apis/cloud/v1"
	myconfig "github.com/oceanbase/ob-operator/pkg/config"
	"github.com/oceanbase/ob-operator/pkg/controllers/observer/model"
	"io/ioutil"
	"k8s.io/klog/v2"
	"net/http"
)

func IsAllOBAgentActive(obAgentList []model.OBAgent, obClusters []cloudv1.Cluster) bool {
	obAgentCurrentReplicas := make(map[string]bool)
	for _, obAgent := range obAgentList {

		if CheckObAgentActive(obAgent) {
			obAgentCurrentReplicas[obAgent.Zone] = true
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

func CheckObAgentActive(obAgent model.OBAgent) bool {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://127.1:8088/metrics/ob/basic", nil)
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
