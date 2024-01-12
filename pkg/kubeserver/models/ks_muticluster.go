package models

import (
	"context"
	"strings"
	"yunion.io/x/jsonutils"
	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/manager"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
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
