package core

import (
	cloudv1 "github.com/oceanbase/ob-operator/apis/cloud/v1"
	"github.com/oceanbase/ob-operator/pkg/controllers/observer/cable"
	observerconst "github.com/oceanbase/ob-operator/pkg/controllers/observer/const"
	"github.com/oceanbase/ob-operator/pkg/controllers/observer/core/converter"
	"github.com/oceanbase/ob-operator/pkg/controllers/observer/sql"
	"k8s.io/klog/v2"
)

func (ctrl *OBClusterCtrl) OBAgentBootstrapReady(statefulApp cloudv1.StatefulApp) error {
	// time out, delete StatefulApp
	status, err := ctrl.CheckTimeoutAndKillForBootstrap(statefulApp)
	if err != nil {
		return err
	}

	if !status {
		subsets := statefulApp.Status.Subsets
		// 获得所有的 obagent
		klog.Infoln("try to get obagent", subsets)
		klog.Infoln("try to get obagent", subsets[0])
		klog.Infoln("try to get obagent", subsets[0].Pods[0])
		klog.Infoln("try to get obagent", subsets[0].Pods[1])
		klog.Infoln("try to get obagent", subsets[0].Pods[0].PodIP)
		klog.Infoln("try to get obagent", subsets[0].Pods[1].PodIP)

		obAgentList := sql.GetOBAgent(subsets[0].Pods[0].PodIP)

		obAgentBootstrapSucceed := converter.IsAllOBAgentActive(obAgentList, ctrl.OBCluster.Spec.Topology)
		if obAgentBootstrapSucceed {
			// update OBServer Pod Readiness
			err = cable.CableReadinessUpdate(subsets)
			if err != nil {
				return err
			}
			// update status
			return ctrl.UpdateOBClusterAndZoneStatus(observerconst.OBAgentReady, "", "")
		}

		klog.Infoln("wait for OBCluster", ctrl.OBCluster.Name, "Bootstraping finish(obAgent)")
	}

	return nil
}
