/*
Copyright (c) 2021 OceanBase
ob-operator is licensed under Mulan PSL v2.
You can use this software according to the terms and conditions of the Mulan PSL v2.
You may obtain a copy of Mulan PSL v2 at:
         http://license.coscl.org.cn/MulanPSL2
THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
See the Mulan PSL v2 for more details.
*/

package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	cloudv1 "github.com/oceanbase/ob-operator/apis/cloud/v1"
	observerconst "github.com/oceanbase/ob-operator/pkg/controllers/observer/const"
	"github.com/oceanbase/ob-operator/pkg/controllers/observer/sql"
	"io/ioutil"
	"k8s.io/klog/v2"
	"net/http"
)

type ConfigsJson struct {
	Configs []Configs `json:"configs"`
}
type Configs struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (ctrl *OBClusterCtrl) CreateUserForObagent(statefulApp cloudv1.StatefulApp) error {
	subsets := statefulApp.Status.Subsets
    podIp := subsets[0].Pods[0].PodIP
    err := sql.CreateUser(podIp, "ocp_monitor", "root")
    klog.Errorln("CreateUser podIP is :", podIp)
    if err != nil {
        return err
    }
    err = sql.GrantPrivilege(podIp, "select", "*", "ocp_monitor")
    if err != nil {
        return err
    }





    // 获得所有的 obagent
	for subsetsIdx, _ := range subsets {
		for _, pod := range subsets[subsetsIdx].Pods {
	//		err := sql.CreateUser(pod.PodIP, "ocp_monitor", "")
    //        klog.Infoln("obagent createuser pod ip", pod.PodIP)
	//		if err != nil {
    //            klog.Errorln("creater user for agent failed," ,err)
	//			return err
	//		}
	//		err = sql.GrantPrivilege(pod.PodIP, "select", "*.*", "ocp_monitor")
	//		if err != nil {
    //            klog.Errorln("grant privilege for agent failed," ,err)
	//			return err
	//		}
			ctrl.ReviseConfig(pod)
		}
	}

	return nil
}

func (ctrl *OBClusterCtrl) ReviseConfig(pod cloudv1.PodStatus) {
	config := ConfigsJson{
		[]Configs{
			{Key: "monagent.ob.monitor.user", Value: "ocp_monitor"},
			{Key: "monagent.ob.monitor.password", Value: ""},
			{Key: "monagent.host.ip", Value: pod.PodIP},
			{Key: "monagent.ob.cluster.name", Value: "ob-test"},
			{Key: "monagent.ob.cluster.id", Value: "1"},
			{Key: "monagent.ob.zone.name", Value: "zone1"}}}

	updateUrl := fmt.Sprintf("http://%s:%d%s", pod.PodIP, observerconst.MonagentPort, observerconst.MonagentUpdateUrl)
	body, _ := json.Marshal(config)
	resp, err := http.Post(updateUrl, "application/json", bytes.NewBuffer(body))

	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			//Failed to read response.
			panic(err)
			klog.Errorln("Fail to read response:", err)
		}
		jsonStr := string(body)
		klog.Errorln("Update config Response: ", jsonStr)
	} else {
		//The status is not Created. print the error.
		klog.Errorln("Get failed with error: ", resp.Status)
	}
}


func SetConfig() string {
	config := `{"configs":[
							{"key":"monagent.ob.monitor.user", "value":"ocp_monitor"},
							{"key":"monagent.ob.monitor.password", "value":"root"},
							{"key":"monagent.host.ip", "value":"10.42.0.178"},
							{"key":"monagent.ob.cluster.name", "value":"ob-test"},
							{"key":"monagent.ob.cluster.id", "value":"1"},
							{"key":"monagent.ob.zone.name", "value":"zone1"}
				]}`
	return config
}
