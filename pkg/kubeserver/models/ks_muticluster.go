package models

import (
	"context"
	"fmt"
	"k8s.io/client-go/tools/clientcmd"
	"strings"
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/ksclient"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/manager"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/sqlchemy"
)

var KsClusterManager *SKsClusterManager

func init() {
	initKSClusterManager()
}

func initKSClusterManager() {
	if KsClusterManager != nil {
		return
	}
	KsClusterManager = &SKsClusterManager{
		SStatusDomainLevelResourceBaseManager: db.NewStatusDomainLevelResourceBaseManager(
			SKsCluster{},
			"ksclusters_tbl",
			"kscluster",
			"ksclusters",
		),
		SSyncableManager: newSyncableManager(),
	}
	manager.RegisterKsClusterManager(KsClusterManager)
	KsClusterManager.SetVirtualObject(KsClusterManager)
	KsClusterManager.SetAlias("kscluster", "ksclusters")
}

// +onecloud:swagger-gen-model-singular=SKsClusterManager
type SKsClusterManager struct {
	db.SStatusDomainLevelResourceBaseManager
	SSyncableK8sBaseResourceManager

	*SSyncableManager
}

type SKsCluster struct {
	db.SStatusDomainLevelResourceBase
	SSyncableK8sBaseResource

	IsSystem bool `nullable:"true" default:"false" list:"admin" create:"optional" json:"is_system"`

	// kubernetes config
	Kubeconfig string `length:"long" nullable:"true" charset:"utf8" create:"optional"`
	// kubernetes api server endpoint
	ApiServer string `width:"256" nullable:"true" charset:"ascii" create:"optional" list:"user"`

	// Version records kubernetes api server version
	Version string `width:"128" charset:"ascii" nullable:"false" create:"optional" list:"user"`

	NodeNum int `json:"node_num" nullable:"false" create:"optional" list:"user"`
}

func (m *SKsClusterManager) InitializeData() error {

	return nil
}

func (m *SKsClusterManager) GetHelloWorld() string {
	return "hello world!"
}

func (m *SKsClusterManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.KsClusterCreateInput) (*api.KsClusterCreateInput, error) {
	sInput, err := m.SStatusDomainLevelResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusDomainLevelResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.StatusDomainLevelResourceCreateInput = sInput
	if input.IsSystem != nil && *input.IsSystem && !db.IsAdminAllowCreate(userCred, m).Result.IsAllow() {
		return nil, httperrors.NewNotSufficientPrivilegeError("non-admin user not allowed to create system object")
	}

	return input, nil
}

func (cluster *SKsCluster) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if err := cluster.SStatusDomainLevelResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
		return err
	}
	input := new(api.KsClusterCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return errors.Wrap(err, "unmarshal cluster create input")
	}
	if input.IsSystem != nil && *input.IsSystem {
		cluster.IsSystem = true
	} else {
		cluster.IsSystem = false
	}
	cluster.Kubeconfig = input.ImportData.Kubeconfig
	cluster.ApiServer = input.ImportData.ApiServer
	return nil
}

func (m *SKsClusterManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input *api.ClusterListInput,
) (*sqlchemy.SQuery, error) {
	q, err := m.SStatusDomainLevelResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StatusDomainLevelResourceListInput)
	if err != nil {
		return nil, err
	}
	//if input.FederatedResourceUsedInput.ShouldDo() {
	//	fedJointMan := GetFedJointClusterManager(input.FederatedKeyword)
	//	if fedJointMan == nil {
	//		return nil, httperrors.NewInputParameterError("federated_keyword %s not found", input.FederatedKeyword)
	//	}
	//	fedMan := fedJointMan.GetFedManager()
	//	fedObj, err := fedMan.FetchByIdOrName(userCred, input.FederatedResourceId)
	//	if err != nil {
	//		return nil, httperrors.NewNotFoundError("federated resource %s %s found error: %v", input.FederatedKeyword, input.FederatedResourceId, err)
	//	}
	//	sq := fedJointMan.Query("cluster_id").Equals("federatedresource_id", fedObj.GetId()).SubQuery()
	//	if *input.FederatedUsed {
	//		q = q.In("id", sq)
	//	} else {
	//		q = q.NotIn("id", sq)
	//	}
	//}
	m.SyncCluster()

	if len(input.ManagerId) != 0 {
		q = q.In("manager_id", input.ManagerId)
	}
	if input.CloudregionId != "" {
		q = q.Equals("cloudregion_id", input.CloudregionId)
	}
	if input.ExternalClusterId != "" {
		q = q.Equals("external_cluster_id", input.ExternalClusterId)
	}
	if input.ExternalCloudClusterId != "" {
		q = q.Equals("external_cloud_cluster_id", input.ExternalCloudClusterId)
	}

	if len(input.Provider) != 0 {
		providers := make([]string, len(input.Provider))
		for i := range input.Provider {
			providers[i] = strings.ToLower(input.Provider[i])
		}
		q = q.In("provider", providers)
	}
	if input.Mode != "" {
		q = q.Equals("mode", input.Mode)
	}
	return q, nil
}

func (c *SKsCluster) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	c.SStatusDomainLevelResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	if err := c.StartClusterCreateTask(ctx, userCred, data.(*jsonutils.JSONDict), ""); err != nil {
		log.Errorf("StartClusterCreateTask error: %v", err)
	}
}

func (c *SKsCluster) StartClusterCreateTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	c.SetStatus(userCred, api.ClusterStatusCreating, "")
	c.MarkSyncing(c, userCred)
	task, err := taskman.TaskManager.NewTask(ctx, "KsClusterCreateTask", c, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (m *SKsClusterManager) SyncCluster() error {
	clusterList, err := ksclient.ListCluster()
	if err != nil {
		fmt.Println(err)
		// return err
	}
	fmt.Println(clusterList.Items)
	q := m.Query()
	//obj, err := db.NewModelObject(MachineManager)
	//if err != nil {
	//	return err
	//}
	objs := make([]SKsCluster, 0)
	err = db.FetchModelObjects(m, q, &objs)
	if err != nil {
		fmt.Println(err)
	}
	var exist bool
	for _, kscluster := range clusterList.Items {
		exist = false
		for _, obj := range objs {
			if kscluster.Name == obj.Name {
				exist = true
				_, err := db.Update(&obj, func() error {
					obj.SetSyncStatus(K8S_SYNC_STATUS_SYNCING)
					obj.SetLastSync(timeutils.UtcNow())
					obj.SetLastSyncEndAt(time.Time{})

					obj.NodeNum = kscluster.Status.NodeCount
					obj.Version = kscluster.Status.KubernetesVersion
					obj.Status = "running"
					obj.ApiServer = GetHost(kscluster.Spec.Connection.KubeConfig)
					return nil
				})
				if err != nil {
					fmt.Println(err)
				}
			}
		}
		if !exist {
			// err = m.TableSpec().InsertOrUpdate(context.Background(), objs[0])

			skscluster := SKsCluster{
				Version:    kscluster.Status.KubernetesVersion,
				Kubeconfig: string(kscluster.Spec.Connection.KubeConfig),
				NodeNum:    kscluster.Status.NodeCount,
				IsSystem:   true,
				ApiServer:  GetHost(kscluster.Spec.Connection.KubeConfig),
			}

			obj, err := db.NewModelObject(m)
			if err != nil {
				fmt.Println(err)
			}
			man := obj.GetModelManager()
			if err != nil {
				return errors.Wrapf(err, "new  model %s", man.Keyword())
			}
			data := jsonutils.NewDict()
			data.Add(jsonutils.NewString(skscluster.Version), "version")
			data.Add(jsonutils.NewString(skscluster.Kubeconfig), "kubeconfig")
			data.Add(jsonutils.NewString(kscluster.Name), "name")
			data.Add(jsonutils.NewInt(int64(skscluster.NodeNum)), "node_num")
			data.Add(jsonutils.NewString(skscluster.ApiServer), "api_server")
			if err := data.Unmarshal(obj); err != nil {
				return httperrors.NewGeneralError(err)
			}
			if err := m.TableSpec().Insert(context.Background(), obj); err != nil {
				return errors.Wrap(err, "insert joint model")
			}

		}
	}
	//fmt.Println(objs)
	return nil
}

func (c *SKsCluster) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	return nil
}

func (c *SKsCluster) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("Cluster delete do nothing")
	return c.RealDelete(ctx, userCred)
}
func (c *SKsCluster) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	/*	if err := c.DeleteAllComponents(ctx, userCred); err != nil {
			return errors.Wrapf(err, "DeleteClusterComponent")
		}
		if err := c.PurgeAllClusterResource(ctx, userCred); err != nil {
			return errors.Wrap(err, "Purge all k8s cluster db resources")
		}
		if err := c.PurgeAllFedResource(ctx, userCred); err != nil {
			return errors.Wrap(err, "Purge all federated cluster resources")
		}*/
	err := ksclient.DeleteCluster(c.Name)
	if err != nil {
		fmt.Println(err)
		// return err
	}
	return c.SStatusDomainLevelResourceBase.Delete(ctx, userCred)
}

func GetHost(kubeconfig []byte) string {
	config, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
	if err != nil {
		fmt.Println(err)
	}
	clientConfig, err := config.ClientConfig()
	if err != nil {
		fmt.Println(err)
	}
	return clientConfig.Host
}
