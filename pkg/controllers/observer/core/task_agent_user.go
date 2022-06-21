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
	"github.com/gin-gonic/gin"
	cloudv1 "github.com/oceanbase/ob-operator/apis/cloud/v1"
	"github.com/oceanbase/ob-operator/pkg/controllers/observer/sql"
	"io/ioutil"
	"k8s.io/klog/v2"
	"net/http"
	"strings"
)

const (
	AgentResponseKey = "agentResponse"
	TraceIdKey       = "traceId"
)

func (ctrl *OBClusterCtrl) CreateUserForObagent(statefulApp cloudv1.StatefulApp) error {
	subsets := statefulApp.Status.Subsets
	podIp := subsets[0].Pods[0].PodIP

	err := sql.CreateUser(podIp, "ocp-monitor", "root")
	if err != nil {
		return err
	}
	err = sql.GrantPrivilege(podIp, "select", "*.*", "ocp-monitor")
	if err != nil {
		return err
	}
	return nil
}

func (ctrl *OBClusterCtrl) ReviseConfig(r *gin.Engine) {
	client := &http.Client{}
	var data = strings.NewReader(SetConfig())
	req, err := http.NewRequest("POST", "http://127.1:8088/api/v1/module/config/update", data)
	if err != nil {
		klog.Errorln("ger new request", err)
	}
	req.Header.Set("content-type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		klog.Errorln("client do", err)
	}
	defer resp.Body.Close()
	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Errorln("read all response body", err)
	}
	klog.Infoln("response body :", bodyText)
}

//curl -X POST                                                                          -X/--request |指定什么命令|
//-H 'content-type:application/json'													-H/--header 自定义头信息传递给服务器
//-d '{"configs":[{"key":"monagent.pipeline.ob.status", "value":"active"}]}'            -d/--data HTTP POST方式传送数据
//-L 'http://127.1:8088/api/v1/module/config/update

func SetConfig() string {
	config := `{"configs":[ 
							{"key":"monagent.ob.monitor.user", "value":"ocp_monitor"},
							{"key":"monagent.ob.monitor.password", "value":"root"},
							{"key":"monagent.host.ip", "value":"10.42.0.178"},
							{"key":"monagent.ob.cluster.name", "value":"ob-test"},
							{"key":"monagent.ob.cluster.id", "value":"1"},
							{"key":"monagent.ob.zone.name", "value":"zone1"},
				]}`
	return config
}
