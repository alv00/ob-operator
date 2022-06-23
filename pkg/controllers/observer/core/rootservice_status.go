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
	"reflect"

	cloudv1 "github.com/oceanbase/ob-operator/apis/cloud/v1"
	"github.com/oceanbase/ob-operator/pkg/controllers/observer/core/converter"
	"github.com/oceanbase/ob-operator/pkg/controllers/observer/model"
	"github.com/oceanbase/ob-operator/pkg/controllers/observer/sql"
	"github.com/oceanbase/ob-operator/pkg/infrastructure/kube"
)

func (ctrl *OBClusterCtrl) UpdateRootServiceStatus(statefulApp cloudv1.StatefulApp) error {
	subsets := statefulApp.Status.Subsets
	// TODO: check owner
	rsName := converter.GenerateRootServiceName(ctrl.OBCluster.Name)
	rsCtrl := NewRootServiceCtrl(ctrl)
	rsCurrent, err := rsCtrl.GetRootServiceByName(ctrl.OBCluster.Namespace, rsName)
	if err != nil {
		return err
	}

	rsList := make([]model.AllVirtualCoreMeta, 0)
	observerList := make([]model.AllServer, 0)
	queryOk := false
	for _, subset := range subsets {
		if queryOk {
			break
		}
		for _, pod := range subset.Pods {
			rsList = sql.GetRootService(pod.PodIP)
			observerList = sql.GetOBServer(pod.PodIP)
			if len(rsList) > 0 && len(observerList) > 0 {
				queryOk = true
				break
			}
		}
	}
	cluster := converter.GetClusterSpecFromOBTopology(ctrl.OBCluster.Spec.Topology)
	rsStatus := converter.RSListToRSStatus(cluster, rsCurrent, rsList, observerList)
	status := reflect.DeepEqual(rsCurrent.Status, rsStatus.Status)
	if !status {
		err = rsCtrl.UpdateRootServiceStatus(rsStatus)
		if err != nil {
			return err
		}
		kube.LogForAppActionStatus(rsStatus.Kind, rsName, "update", rsStatus)
	}
	return nil
}
