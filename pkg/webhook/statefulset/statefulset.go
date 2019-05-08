package statefulset

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	"k8s.io/api/admission/v1beta1"
	apps "k8s.io/api/apps/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func AdmitStatefulSets(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	glog.Infof("admit statefulsets")

	setResource := metav1.GroupVersionResource{Group: "apps", Version: "v1beta1", Resource: "statefulsets"}
	if ar.Request.Resource != setResource {
		err := fmt.Errorf("expect resource to be %s", setResource)
		glog.Errorf("%v", err)
		return util.ToAdmissionResponse(err)
	}

	cli, _, err := util.GetNewClient()
	if err != nil {
		glog.Errorf("failed to get kubernetes Clientset: %v", err)
		return util.ToAdmissionResponse(err)
	}

	name := ar.Request.Name
	namespace := ar.Request.Namespace

	raw := ar.Request.OldObject.Raw
	set := apps.StatefulSet{}
	deserializer := util.GetCodec()
	if _, _, err := deserializer.Decode(raw, nil, &set); err != nil {
		glog.Error(err)
		return util.ToAdmissionResponse(err)
	}

	tc, err := cli.PingcapV1alpha1().TidbClusters(namespace).Get(set.Labels[label.InstanceLabelKey], metav1.GetOptions{})
	if err != nil {
		glog.Errorf("fail to fetch tidbcluster info namespace %s clustername(instance) %s err %v", namespace, set.Labels[label.InstanceLabelKey], err)
		return util.ToAdmissionResponse(err)
	}

	if set.Labels[label.ComponentLabelKey] == "tidb" {
		protect, ok := tc.Annotations[label.AnnTiDBPartition]

		if ok {
			partition, err := strconv.ParseInt(protect, 10, 32)
			if err != nil {
				glog.Errorf("fail to convert protect to int namespace %s name %s err %v", namespace, name, err)
				return util.ToAdmissionResponse(err)
			}

			if (*set.Spec.UpdateStrategy.RollingUpdate.Partition) <= int32(partition) && tc.Status.TiDB.Phase == v1alpha1.UpgradePhase {
				glog.Infof("set has been protect by annotations name %s namespace %s", name, namespace)
				return util.ToAdmissionResponse(k8serrors.NewConflict(schema.GroupResource{Resource: "statefulset"}, "webhook", errors.New("protect by annotation")))
			}
		}
	}

	return util.ToAdmissionResponseSuccess()
}
