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
	"time"

	"github.com/pkg/errors"
	"k8s.io/klog/v2"

	cloudv1 "github.com/oceanbase/ob-operator/apis/cloud/v1"
	"github.com/oceanbase/ob-operator/pkg/controllers/observer/cable"
	observerconst "github.com/oceanbase/ob-operator/pkg/controllers/observer/const"
	"github.com/oceanbase/ob-operator/pkg/controllers/observer/core/converter"
	"github.com/oceanbase/ob-operator/pkg/controllers/observer/sql"
)

func (ctrl *OBClusterCtrl) AddOBServer(clusterIP, zoneName, podIP string, statefulApp cloudv1.StatefulApp) error {
	clusterStatus := converter.GetClusterStatusFromOBTopologyStatus(ctrl.OBCluster.Status.Topology)
	for _, zone := range clusterStatus.Zone {
		if zone.ZoneStatus == observerconst.OBServerAdd {
			// judge timeout
			lastTransitionTimestamp := clusterStatus.LastTransitionTime.Unix()
			nowTimestamp := time.Now().Unix()
			if nowTimestamp-lastTransitionTimestamp > observerconst.AddServerTimeout {
				klog.Infoln("add server timeout, need delete", zoneName, podIP)
				err := ctrl.DelPodFromStatefulAppByIP(zoneName, podIP, statefulApp)
				if err != nil {
					return err
				}
				return ctrl.UpdateOBClusterAndZoneStatus(observerconst.ClusterReady, zoneName, observerconst.OBZoneReady)
			}
			return nil
		}
	}
	for _, zone := range clusterStatus.Zone {
		// an IP is already in the process of being added
		// execute serially, one IP at a time
		if zone.Name == zoneName && zone.ZoneStatus != observerconst.OBServerAdd {
			// add server and update obagent config
			go func() {
				ctrl.AddOBServerExecuter(clusterIP, zoneName, podIP, statefulApp)
				ctrl.ReviseConfig(podIP)
			}()
			// update status
			return ctrl.UpdateOBClusterAndZoneStatus(observerconst.ScaleUP, zoneName, observerconst.OBServerAdd)
		}
	}
	return nil
}

func (ctrl *OBClusterCtrl) AddOBServerExecuter(clusterIP, zoneName, podIP string, statefulApp cloudv1.StatefulApp) {
	klog.Infoln("begin add OBServer", zoneName, podIP)

	// check cable status
	err := cable.CableStatusCheckExecuter(podIP)
	if err != nil {
		// kill pod
		_ = ctrl.DelPodFromStatefulAppByIP(zoneName, podIP, statefulApp)
		_ = ctrl.UpdateOBClusterAndZoneStatus(observerconst.ClusterReady, zoneName, observerconst.OBZoneReady)
	}

	// get rs
	rsName := converter.GenerateRootServiceName(ctrl.OBCluster.Name)
	rsCtrl := NewRootServiceCtrl(ctrl)
	rsCurrent, err := rsCtrl.GetRootServiceByName(ctrl.OBCluster.Namespace, rsName)
	if err != nil {
		// kill pod
		_ = ctrl.DelPodFromStatefulAppByIP(zoneName, podIP, statefulApp)
		_ = ctrl.UpdateOBClusterAndZoneStatus(observerconst.ClusterReady, zoneName, observerconst.OBZoneReady)
	}

	// generate rsList
	rsList := cable.GenerateRSListFromRootServiceStatus(rsCurrent.Status.Topology)

	// generate start args
	obServerStartArgs := cable.GenerateOBServerStartArgs(ctrl.OBCluster, zoneName, rsList)

	// check OBServer is already running, for OBServer Scale UP
	err = cable.OBServerStatusCheckExecuter(ctrl.OBCluster.Name, podIP)
	// nil is OBServer is already running
	if err != nil {
		cable.OBServerStartExecuter(podIP, obServerStartArgs)
		err = TickerOBServerStatusCheck(ctrl.OBCluster.Name, podIP)
		if err != nil {
			// kill pod
			_ = ctrl.DelPodFromStatefulAppByIP(zoneName, podIP, statefulApp)
			_ = ctrl.UpdateOBClusterAndZoneStatus(observerconst.ClusterReady, zoneName, observerconst.OBZoneReady)
		}
	}

	// add server
	err = sql.AddServer(clusterIP, zoneName, podIP)
	if err != nil {
		// kill pod
		_ = ctrl.DelPodFromStatefulAppByIP(zoneName, podIP, statefulApp)
		_ = ctrl.UpdateOBClusterAndZoneStatus(observerconst.ClusterReady, zoneName, observerconst.OBZoneReady)
	}

	err = TickerOBServerStatusCheckFromDB(clusterIP, podIP)
	if err != nil {
		// kill pod
		_ = ctrl.DelPodFromStatefulAppByIP(zoneName, podIP, statefulApp)
		_ = ctrl.UpdateOBClusterAndZoneStatus(observerconst.ClusterReady, zoneName, observerconst.OBZoneReady)
	}

	// update OBServer Pod Readiness
	err = cable.CableReadinessUpdateExecuter(podIP)
	if err != nil {
		// kill pod
		_ = ctrl.DelPodFromStatefulAppByIP(zoneName, podIP, statefulApp)
		_ = ctrl.UpdateOBClusterAndZoneStatus(observerconst.ClusterReady, zoneName, observerconst.OBZoneReady)
	}

	klog.Infoln("add OBServer finish", zoneName, podIP)

	// update status
	_ = ctrl.UpdateOBClusterAndZoneStatus(observerconst.ClusterReady, zoneName, observerconst.OBZoneReady)
}

func TickerOBServerStatusCheck(clusterName, podIP string) error {
	tick := time.Tick(observerconst.TickPeriodForOBServerStatusCheck)
	var num int
	var err error
	for {
		select {
		case <-tick:
			if num > observerconst.TickNumForOBServerStatusCheck {
				return errors.New("observer starting timeout")
			}
			num = num + 1
			err = cable.OBServerStatusCheckExecuter(clusterName, podIP)
			if err == nil {
				return err
			}
		}
	}

}

func TickerOBServerStatusCheckFromDB(clusterIP string, podIP string) error {
	tick := time.Tick(observerconst.TickPeriodForOBServerStatusCheck)
	var num int
	for {
		select {
		case <-tick:
			if num > observerconst.TickNumForOBServerStatusCheck {
				return errors.New("observer starting timeout")
			}
			num = num + 1
			obServerList := sql.GetOBServer(clusterIP)
			klog.Infoln("TickerOBServerStatusCheck: obServerList", obServerList)
			for _, obServer := range obServerList {
				klog.Infoln("TickerOBServerStatusCheck: obServer.SvrIP", obServer.SvrIP)
				klog.Infoln("TickerOBServerStatusCheck: podIP", podIP)
				if obServer.SvrIP == podIP {
					if obServer.Status == observerconst.OBServerActive && obServer.StartServiceTime > 0 {
						return nil
					}
				}
			}
		}
	}

}
