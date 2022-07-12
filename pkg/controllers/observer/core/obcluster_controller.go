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
	"context"
	"k8s.io/klog/v2"

	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cloudv1 "github.com/oceanbase/ob-operator/apis/cloud/v1"
	myconfig "github.com/oceanbase/ob-operator/pkg/config"
	observerconst "github.com/oceanbase/ob-operator/pkg/controllers/observer/const"
	"github.com/oceanbase/ob-operator/pkg/controllers/observer/core/judge"
	"github.com/oceanbase/ob-operator/pkg/infrastructure/kube/resource"
)

// OBClusterReconciler reconciles a OBCluster object
type OBClusterReconciler struct {
	CRClient client.Client
	Scheme   *runtime.Scheme
	// https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/events/event.go
	Recorder record.EventRecorder
}

type OBClusterCtrl struct {
	Resource  *resource.Resource
	OBCluster cloudv1.OBCluster
}

type OBClusterCtrlOperator interface {
	OBClusterCoordinator() (ctrl.Result, error)
}

// +kubebuilder:rbac:groups=cloud.oceanbase.com,resources=obclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloud.oceanbase.com,resources=obclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cloud.oceanbase.com,resources=obclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=cloud.oceanbase.com,resources=rootservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloud.oceanbase.com,resources=rootservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cloud.oceanbase.com,resources=rootservices/finalizers,verbs=update
// +kubebuilder:rbac:groups=cloud.oceanbase.com,resources=obzones,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloud.oceanbase.com,resources=obzones/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cloud.oceanbase.com,resources=obzones/finalizers,verbs=update
// +kubebuilder:rbac:groups=cloud.oceanbase.com,resources=statefulapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloud.oceanbase.com,resources=statefulapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cloud.oceanbase.com,resources=statefulapps/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=services/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *OBClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the CR instance
	instance := &cloudv1.OBCluster{}
	err := r.CRClient.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if kubeerrors.IsNotFound(err) {
			// Object not found, return.
			// Created objects are automatically garbage collected.
			return reconcile.Result{}, nil
		}
		// Error reading the object, requeue the request.
		return reconcile.Result{}, err
	}
	// custom logic
	obClusterCtrl := NewOBServerCtrl(r.CRClient, r.Recorder, *instance)
	return obClusterCtrl.OBClusterCoordinator()
}

func NewOBServerCtrl(client client.Client, recorder record.EventRecorder, obCluster cloudv1.OBCluster) OBClusterCtrlOperator {
	ctrlResource := resource.NewResource(client, recorder)
	return &OBClusterCtrl{
		Resource:  ctrlResource,
		OBCluster: obCluster,
	}
}

func (ctrl *OBClusterCtrl) OBClusterCoordinator() (ctrl.Result, error) {
	var newClusterStatus bool
	statefulApp := &cloudv1.StatefulApp{}
	statefulApp, newClusterStatus = ctrl.IsNewCluster(*statefulApp)

	// is new cluster
	if newClusterStatus {
		err := ctrl.NewCluster(*statefulApp)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// OBCluster control-plan
	err := ctrl.OBClusterEffector(*statefulApp)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (ctrl *OBClusterCtrl) OBClusterEffector(statefulApp cloudv1.StatefulApp) error {
	var err error
	obClusterStatus := ctrl.OBCluster.Status.Status
	klog.Infoln("OBClusterEffector: ctrl.OBCluster ", ctrl.OBCluster)
	klog.Infoln("OBClusterEffector: StatefulApp ", statefulApp)
	switch obClusterStatus {
	case observerconst.TopologyPrepareing:
		// OBCluster is not ready
		klog.Info("obcluster is preparing")
		err = ctrl.TopologyPrepareingEffector(statefulApp)
	case observerconst.TopologyNotReady:
		// OBCluster is not ready
		klog.Info("obcluster is not ready")
		err = ctrl.TopologyNotReadyEffector(statefulApp)
	case observerconst.TopologyReady:
		// OBCluster is ready
		klog.Info("obcluster is ready")
		err = ctrl.TopologyReadyEffector(statefulApp)
	}
	return err
}

func (ctrl *OBClusterCtrl) TopologyPrepareingEffector(statefulApp cloudv1.StatefulApp) error {
	var err error

	for _, clusterStatus := range ctrl.OBCluster.Status.Topology {
		if clusterStatus.Cluster == myconfig.ClusterName {
			switch clusterStatus.ClusterStatus {
			case observerconst.ResourcePrepareing:
				// StatefulApp is creating
				err = ctrl.ResourcePrepareingEffectorForBootstrap(statefulApp)
			case observerconst.ResourceReady:
				// StatefulApp is ready
				err = ctrl.ResourceReadyEffectorForBootstrap(statefulApp)
			case observerconst.OBServerPrepareing:
				// OBServer is staring
				err = ctrl.OBServerPrepareingEffectorForBootstrap(statefulApp)
			case observerconst.OBServerReady:
				// OBServer is running
				err = ctrl.OBServerReadyEffectorForBootstrap(statefulApp)
			case observerconst.OBClusterBootstraping:
				// OBCluster Bootstraping
				err = ctrl.OBClusterBootstraping(statefulApp)
			case observerconst.OBClusterBootstrapReady:
				// OBCluster Bootstrap ready
				err = ctrl.OBClusterBootstrapReady(statefulApp)
			case observerconst.OBClusterReady:
				// OBCluster bootstrap succeeded
				err = ctrl.OBClusterReadyForStep(observerconst.StepBootstrap, statefulApp)
				if err == nil {
					err = ctrl.CreateUserForObproxy(statefulApp)
					klog.Infoln("preparation for obagent")
					err = ctrl.CreateUserForObagent(statefulApp)
					err = ctrl.ReviseAllOBAgentConfig(statefulApp)
				}
			}
		}
	}
	return err
}

func (ctrl *OBClusterCtrl) TopologyNotReadyEffector(statefulApp cloudv1.StatefulApp) error {
	var err error
	for _, clusterStatus := range ctrl.OBCluster.Status.Topology {
		if clusterStatus.Cluster == myconfig.ClusterName {
			switch clusterStatus.ClusterStatus {
			case observerconst.ScaleUP:
				// OBServer Scale UP
				err = ctrl.OBServerScaleUPByZone(statefulApp)
			case observerconst.ScaleDown:
				// OBServer Scale Down
				err = ctrl.OBServerScaleDownByZone(statefulApp)
			}
		}
	}
	return err
}

func (ctrl *OBClusterCtrl) TopologyReadyEffector(statefulApp cloudv1.StatefulApp) error {
	// check parameter and version in obcluster, set parameter when modified
	klog.Info("obcluster ready effector")
	ctrl.CheckAndSetParameters()

	// check version update
	versionIsModified, err := judge.VersionIsModified(ctrl.OBCluster.Spec.Tag, statefulApp)
	if err != nil {
		return err
	}
	if versionIsModified {
		// TODO: support version update
		klog.Errorln("version update is not supported yet")
		return nil
	}

	// check resource modified
	resourcesIsModified, err := judge.ResourcesIsModified(ctrl.OBCluster.Spec.Topology, ctrl.OBCluster, statefulApp)
	if err != nil {
		return err
	}
	if resourcesIsModified {
		// TODO: support resource change
		klog.Errorln("resource changes is not supported yet")
		return nil
	}

	// check zone number modified
	zoneScaleStatus, err := judge.ZoneNumberIsModified(ctrl.OBCluster.Spec.Topology, statefulApp)
	if err != nil {
		return err
	}
	switch zoneScaleStatus {
	case observerconst.ScaleUP:
		// TODO: support Zone Scale UP
		klog.Errorln("Zone scale up is not supported yet")
	case observerconst.ScaleDown:
		// TODO: support Zone Scale Down
		klog.Errorln("Zone scale down is not supported yet")
	case observerconst.Maintain:
		err = ctrl.OBServerCoordinator(statefulApp)
		if err != nil {
			return err
		}
	}

	return nil
}
