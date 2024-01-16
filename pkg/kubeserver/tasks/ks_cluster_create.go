package tasks

import (
	"context"
	"fmt"
	clusterv1alpha1 "kubesphere.io/api/cluster/v1alpha1"
	"yunion.io/x/jsonutils"
	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/ksclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
)

func init() {
	taskman.RegisterTask(KsClusterCreateTask{})
}

type KsClusterCreateTask struct {
	taskman.STask
}

func (t *KsClusterCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	// client := GetKsClient()
	fmt.Println("hello world!", clusterv1alpha1.Cluster{})
	data.GetString("name")
	input := api.KsClusterCreateInput{}
	if err := t.GetParams().Unmarshal(&input); err != nil {
		return
	}
	res, err := ksclient.CreateCluster(input)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(res)
}
