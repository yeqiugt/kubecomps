package models

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/embed"
	"yunion.io/x/kubecomps/pkg/kubeserver/templates/components"
)

const (
	ThanosReleaseName = "thanos"
)

var (
	ThanosComponentManager *SThanosComponentManager
)

func init() {
	ThanosComponentManager = NewThanosComponentManager()
	ComponentManager.RegisterDriver(newComponentDriverThanos())
}

type SThanosComponentManager struct {
	SComponentManager
	HelmComponentManager
}

type SThanosComponent struct {
	SComponent
}

func NewThanosComponentManager() *SThanosComponentManager {
	man := new(SThanosComponentManager)
	man.SComponentManager = *NewComponentManager(SThanosComponent{},
		"kubecomponentthanos",
		"kubecomponentthanoses",
	)
	man.HelmComponentManager = *NewHelmComponentManager(MonitorNamespace, ThanosReleaseName, embed.THANOS_3_2_2_TGZ)
	man.SetVirtualObject(man)
	return man
}

type componentDriverThanos struct {
	helmComponentDriver
}

func newComponentDriverThanos() IComponentDriver {
	return &componentDriverThanos{
		helmComponentDriver: newHelmComponentDriver(api.ClusterComponentThanos, ThanosComponentManager),
	}
}

func (c componentDriverThanos) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, input *api.ComponentCreateInput) error {
	return c.validateSetting(ctx, userCred, cluster, input.Thanos)
}

func (c componentDriverThanos) validateSetting(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, conf *api.ComponentSettingThanos) error {
	if conf == nil {
		return httperrors.NewNotEmptyError("thanos config is empty")
	}

	if err := validateObjectStore(ctx, &conf.ObjectStoreConfig); err != nil {
		return err
	}

	if conf.Store.Storage.Enabled {
		if err := c.validateStorage(userCred, cluster, &conf.Store.Storage); err != nil {
			return errors.Wrap(err, "validate storegateway storage")
		}
	}

	if conf.Compactor.Storage.Enabled {
		if err := c.validateStorage(userCred, cluster, &conf.Compactor.Storage); err != nil {
			return errors.Wrap(err, "validate compactor storage")
		}
	}

	return nil
}

func (c componentDriverThanos) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, input *api.ComponentUpdateInput) error {
	return c.validateSetting(ctx, userCred, cluster, input.Thanos)
}

func (c componentDriverThanos) GetCreateSettings(input *api.ComponentCreateInput) (*api.ComponentSettings, error) {
	if input.ComponentSettings.Namespace == "" {
		input.ComponentSettings.Namespace = MonitorNamespace
	}
	return &input.ComponentSettings, nil
}

func (c componentDriverThanos) GetUpdateSettings(oldSetting *api.ComponentSettings, input *api.ComponentUpdateInput) (*api.ComponentSettings, error) {
	oldSetting.Thanos = input.Thanos
	return oldSetting, nil
}

func (c componentDriverThanos) DoEnable(cluster *SCluster, setting *api.ComponentSettings) error {
	return ThanosComponentManager.CreateHelmResource(cluster, setting)
}

func (c componentDriverThanos) DoDisable(cluster *SCluster, setting *api.ComponentSettings) error {
	return ThanosComponentManager.DeleteHelmResource(cluster, setting)
}

func (c componentDriverThanos) DoUpdate(cluster *SCluster, setting *api.ComponentSettings) error {
	return ThanosComponentManager.UpdateHelmResource(cluster, setting)
}

func (c componentDriverThanos) FetchStatus(cluster *SCluster, comp *SComponent, status *api.ComponentsStatus) error {
	if status.Thanos == nil {
		status.Thanos = new(api.ComponentStatus)
	}
	c.InitStatus(comp, status.Thanos)
	return nil
}

func (m SThanosComponentManager) CreateHelmResource(cluster *SCluster, setting *api.ComponentSettings) error {
	vals, err := m.GetHelmValues(cluster, setting)
	if err != nil {
		return errors.Wrap(err, "get helm config values")
	}
	return m.HelmComponentManager.CreateHelmResource(cluster, vals)
}

func (m SThanosComponentManager) DeleteHelmResource(cluster *SCluster, setting *api.ComponentSettings) error {
	return m.HelmComponentManager.DeleteHelmResource(cluster)
}

func (m SThanosComponentManager) UpdateHelmResource(cluster *SCluster, setting *api.ComponentSettings) error {
	vals, err := m.GetHelmValues(cluster, setting)
	if err != nil {
		return errors.Wrap(err, "get helm config values")
	}
	return m.HelmComponentManager.UpdateHelmResource(cluster, vals)
}

func (m SThanosComponentManager) GetHelmValues(cluster *SCluster, setting *api.ComponentSettings) (map[string]interface{}, error) {
	imgRepo, err := m.GetImageRepository(cluster, setting)
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster %s repo", cluster.GetName())
	}
	mi := func(name, tag string) components.Image2 {
		parts := strings.Split(imgRepo.Url, "/")
		reg, repo := parts[0], parts[1]
		return components.Image2{
			Registry:   reg,
			Repository: fmt.Sprintf("%s/%s", repo, name),
			Tag:        tag,
		}
	}

	input := setting.Thanos

	conf := components.Thanos{
		Image:         mi("thanos", "v0.22.0"),
		ClusterDomain: input.ClusterDomain,
		ObjStoreConfig: components.ThanosObjectStoreConfig{
			Type: "s3",
			Config: components.ThanosObjectStoreConfigConfig{
				ObjectStoreConfig: input.ObjectStoreConfig,
				SignatureVersion2: true,
			},
		},
		Query: components.ThanosQuery{
			Enabled: true,
			DnsDiscovery: components.ThanosQueryDnsDiscovery{
				Enabled:           true,
				SidecarsService:   input.Query.DnsDiscovery.SidecarsService,
				SidecarsNamespace: input.Query.DnsDiscovery.SidecarsNamespace,
			},
			Stores:       input.Query.Stores,
			ReplicaCount: 1,
		},
		Storegateway: components.ThanosStoregateway{
			Enabled: true,
			Persistence: components.Storage{
				Enabled: false,
			},
		},
		Compactor: components.ThanosCompactor{
			Enabled: true,
			Persistence: components.Storage{
				Enabled: false,
			},
		},
	}

	if input.Store.Storage.Enabled {
		storePvc, err := components.NewPVCStorage(&input.Store.Storage)
		if err != nil {
			return nil, errors.Wrap(err, "new storegateway pvc")
		}
		conf.Storegateway.Persistence = *storePvc
	}

	if input.Compactor.Storage.Enabled {
		compactorPvc, err := components.NewPVCStorage(&input.Compactor.Storage)
		if err != nil {
			return nil, errors.Wrap(err, "new compactor pvc")
		}
		conf.Compactor.Persistence = *compactorPvc
	}

	if cluster.IsSystemCluster() {
		commonConf := getSystemComponentCommonConfig(
			components.CommonConfig{
				Enabled: true,
				Resources: &api.HelmValueResources{
					Limits:   api.NewHelmValueResource("1", "1024Mi"),
					Requests: api.NewHelmValueResource("0.01", "10Mi"),
				},
			},
			false, false)
		conf.Query.CommonConfig = commonConf
		conf.Storegateway.CommonConfig = commonConf
		conf.Compactor.CommonConfig = commonConf
	}

	return components.GenerateHelmValues(conf), nil
}
