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

const (
	AgentResponseKey = "agentResponse"
	TraceIdKey       = "traceId"
)

type Config struct {
	User        string `json:"user"`
	Password    string `json:"password"`
	HostIp      string `json:"hostIp"`
	ClusterName string `json:"clusterName"`
	ClusterId   string `json:"clusterId"`
	ZoneName    string `json:"zoneName"`
}

func (ctrl *OBClusterCtrl) CreateUserForObagent(statefulApp cloudv1.StatefulApp) error {
	subsets := statefulApp.Status.Subsets
	// 获得所有的 obagent
	klog.Infoln("try to get obagent", subsets)
	klog.Infoln("try to get obagent", subsets[0])
	klog.Infoln("try to get obagent", subsets[0].Pods[0])
	klog.Infoln("try to get obagent", subsets[0].Pods[1])
	klog.Infoln("try to get obagent", subsets[0].Pods[0].PodIP)
	klog.Infoln("try to get obagent", subsets[0].Pods[1].PodIP)
	podIp := subsets[0].Pods[0].PodIP

	err := sql.CreateUser(podIp, "ocp_monitor", "root")
	if err != nil {
		return err
	}
	err = sql.GrantPrivilege(podIp, "select", "*.*", "ocp_monitor")
	if err != nil {
		return err
	}
	ctrl.ReviseConfig(podIp)
	return nil
}

func (ctrl *OBClusterCtrl) ReviseConfig(podIp string) {
	config := Config{
		User:        "ocp_monitor",
		Password:    "root",
		HostIp:      podIp,
		ClusterName: "ob-test",
		ClusterId:   "1",
		ZoneName:    "zone1",
	}

	updateUrl := fmt.Sprintf("http://%s:%d%s", podIp, observerconst.MonagentPort, observerconst.MonagentUpdateUrl)
	body, _ := json.Marshal(config)
	resp, err := http.Post(updateUrl, "application/json", bytes.NewBuffer(body))

	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
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
