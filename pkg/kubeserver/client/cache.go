package client

import (
	"context"
	"fmt"
	"time"

	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	apps "k8s.io/client-go/listers/apps/v1"
	autoscalingv1 "k8s.io/client-go/listers/autoscaling/v2beta2"
	batch "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/listers/core/v1"
	rbac "k8s.io/client-go/listers/rbac/v1"
	storage "k8s.io/client-go/listers/storage/v1"
	cache "k8s.io/client-go/tools/cache"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	kapi "yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/manager"
)

var (
	eventWorkMan = appsrv.NewWorkerManager("K8SEventHandlerWorkerManager", 4, 10240, true)
)

type CacheFactory struct {
	stopChan               chan struct{}
	sharedInformerFactory  informers.SharedInformerFactory
	dynamicInformerFactory dynamicinformer.DynamicSharedInformerFactory
	bidirectionalSync      bool
	gvkrs                  []api.ResourceMap
	genericInformers       map[string]informers.GenericInformer
}

func accessCheck(cli kubernetes.Interface, namespace string, verb string, group string, resource string) (bool, error) {
	authCli := cli.AuthorizationV1()
	sar := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      verb,
				Group:     group,
				Resource:  resource,
			},
		},
	}
	args := fmt.Sprintf("%s/%s/%s/%s", namespace, verb, group, resource)
	response, err := authCli.SelfSubjectAccessReviews().Create(context.Background(), sar, metav1.CreateOptions{})
	if err != nil {
		return false, errors.Wrapf(err, "SelfSubjectAccessReviews %s", args)
	}
	if response.Status.Allowed {
		return true, nil
	}
	return false, errors.Errorf("Not allowed %s: %s", args, response.Status.Reason)
}

func buildCacheController(
	cluster manager.ICluster,
	client *kubernetes.Clientset,
	dynamicClient dynamic.Interface,
	resources []api.ResourceMap,
) (*CacheFactory, error) {
	stop := make(chan struct{})
	cacheF := &CacheFactory{
		stopChan:          stop,
		bidirectionalSync: false,
		genericInformers:  make(map[string]informers.GenericInformer),
	}
	sharedInformerFactory := informers.NewSharedInformerFactory(client, 0)
	dynamicInformerFactory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, 0)

	// Start all Resources defined in KindToResourceMap
	informerSyncs := make([]cache.InformerSynced, 0)
	for _, value := range api.KindToResourceMap {
		if utils.IsInStringArray(value.GroupVersionResourceKind.Kind, api.KindHandledByDynamic) {
			continue
		}
		genericInformer, err := sharedInformerFactory.ForResource(value.GroupVersionResourceKind.GroupVersionResource)
		if err != nil {
			return nil, err
		}
		// test watch permissions
		allowed, err := accessCheck(client, "", "watch", value.GroupVersionResourceKind.Group, value.GroupVersionResourceKind.Resource)
		if !allowed {
			return nil, errors.Wrap(err, "watch accessCheck")
		}
		resMan := cluster.GetK8sResourceManager(value.GroupVersionResourceKind.Kind)
		if resMan != nil {
			// register informer event handler
			genericInformer.Informer().AddEventHandler(newEventHandler(cacheF, cluster, resMan))
			cacheF.genericInformers[value.GroupVersionResourceKind.Kind] = genericInformer
		}
		informerSyncs = append(informerSyncs, genericInformer.Informer().HasSynced)
		go genericInformer.Informer().Run(stop)
	}

	// Start all dynamic rest mapper resource
	for _, resource := range resources {
		res := resource.GroupVersionResourceKind.GroupVersionResource
		kind := resource.GroupVersionResourceKind.Kind
		if !utils.IsInStringArray(kind, api.KindHandledByDynamic) {
			continue
		}
		resMan := cluster.GetK8sResourceManager(kind)
		dynamicInformer := dynamicInformerFactory.ForResource(res)
		if resMan != nil {
			// test watch permissions
			allowed, err := accessCheck(client, "", "watch", res.Group, res.Resource)
			if !allowed {
				return nil, errors.Wrap(err, "watch accessCheck")
			}
			// register informer event handler
			dynamicInformer.Informer().AddEventHandler(newEventHandler(cacheF, cluster, resMan))
			cacheF.genericInformers[kind] = dynamicInformer
		}
		informerSyncs = append(informerSyncs, dynamicInformer.Informer().HasSynced)
		go dynamicInformer.Informer().Run(stop)
	}

	// NOTE: Informer().Run has been called, so don't call Factory.Start again
	// sharedInformerFactory.Start(stop)
	// dynamicInformerFactory.Start(stop)

	log.Infof("[Start] WaitForCacheSync for cluster %s(%s)", cluster.GetName(), cluster.GetId())
	if !cache.WaitForCacheSync(stop, informerSyncs...) {
		log.Errorf("[End] WaitForCacheSync for cluster %s(%s) not done", cluster.GetName(), cluster.GetId())
		return nil, errors.Errorf("informers not synced")
	}
	log.Infof("[End] WaitForCacheSync for cluster %s(%s)", cluster.GetName(), cluster.GetId())
	cacheF.sharedInformerFactory = sharedInformerFactory
	cacheF.dynamicInformerFactory = dynamicInformerFactory
	cacheF.gvkrs = resources
	return cacheF, nil
}

func (c *CacheFactory) PodLister() v1.PodLister {
	return c.sharedInformerFactory.Core().V1().Pods().Lister()
}

func (c *CacheFactory) EventLister() v1.EventLister {
	return c.sharedInformerFactory.Core().V1().Events().Lister()
}

func (c *CacheFactory) ConfigMapLister() v1.ConfigMapLister {
	return c.sharedInformerFactory.Core().V1().ConfigMaps().Lister()
}

func (c *CacheFactory) SecretLister() v1.SecretLister {
	return c.sharedInformerFactory.Core().V1().Secrets().Lister()
}

func (c *CacheFactory) DeploymentLister() apps.DeploymentLister {
	return c.sharedInformerFactory.Apps().V1().Deployments().Lister()
}

func (c *CacheFactory) DaemonSetLister() apps.DaemonSetLister {
	return c.sharedInformerFactory.Apps().V1().DaemonSets().Lister()
}

func (c *CacheFactory) StatefulSetLister() apps.StatefulSetLister {
	return c.sharedInformerFactory.Apps().V1().StatefulSets().Lister()
}

func (c *CacheFactory) NodeLister() v1.NodeLister {
	return c.sharedInformerFactory.Core().V1().Nodes().Lister()
}

func (c *CacheFactory) EndpointLister() v1.EndpointsLister {
	return c.sharedInformerFactory.Core().V1().Endpoints().Lister()
}

func (c *CacheFactory) HPALister() autoscalingv1.HorizontalPodAutoscalerLister {
	return c.sharedInformerFactory.Autoscaling().V2beta2().HorizontalPodAutoscalers().Lister()
}

func (c *CacheFactory) GetGVKR(kindName string) *api.ResourceMap {
	for _, r := range c.gvkrs {
		if r.GroupVersionResourceKind.Kind == kindName || r.GroupVersionResourceKind.Resource == kindName {
			return &r
		}
	}
	log.Errorf("Not find by kind: %q", kindName)
	return nil
}

func (c *CacheFactory) IngressLister() cache.GenericLister {
	// return c.sharedInformerFactory.Extensions().V1beta1().Ingresses().Lister()
	gvkr := c.GetGVKR(kapi.KindNameIngress)
	return c.dynamicInformerFactory.ForResource(gvkr.GroupVersionResourceKind.GroupVersionResource).Lister()
}

func (c *CacheFactory) CronJobLister() cache.GenericLister {
	gvkr := c.GetGVKR(kapi.KindNameCronJob)
	return c.dynamicInformerFactory.ForResource(gvkr.GroupVersionResourceKind.GroupVersionResource).Lister()
}

func (c *CacheFactory) ServiceLister() v1.ServiceLister {
	return c.sharedInformerFactory.Core().V1().Services().Lister()
}

func (c *CacheFactory) LimitRangeLister() v1.LimitRangeLister {
	return c.sharedInformerFactory.Core().V1().LimitRanges().Lister()
}

func (c *CacheFactory) NamespaceLister() v1.NamespaceLister {
	return c.sharedInformerFactory.Core().V1().Namespaces().Lister()
}

func (c *CacheFactory) ReplicationControllerLister() v1.ReplicationControllerLister {
	return c.sharedInformerFactory.Core().V1().ReplicationControllers().Lister()
}

func (c *CacheFactory) ReplicaSetLister() apps.ReplicaSetLister {
	return c.sharedInformerFactory.Apps().V1().ReplicaSets().Lister()
}

func (c *CacheFactory) JobLister() batch.JobLister {
	return c.sharedInformerFactory.Batch().V1().Jobs().Lister()
}

func (c *CacheFactory) PVLister() v1.PersistentVolumeLister {
	return c.sharedInformerFactory.Core().V1().PersistentVolumes().Lister()
}

func (c *CacheFactory) PVCLister() v1.PersistentVolumeClaimLister {
	return c.sharedInformerFactory.Core().V1().PersistentVolumeClaims().Lister()
}

func (c *CacheFactory) StorageClassLister() storage.StorageClassLister {
	return c.sharedInformerFactory.Storage().V1().StorageClasses().Lister()
}

func (c *CacheFactory) ResourceQuotaLister() v1.ResourceQuotaLister {
	return c.sharedInformerFactory.Core().V1().ResourceQuotas().Lister()
}

func (c *CacheFactory) RoleLister() rbac.RoleLister {
	return c.sharedInformerFactory.Rbac().V1().Roles().Lister()
}

func (c *CacheFactory) ClusterRoleLister() rbac.ClusterRoleLister {
	return c.sharedInformerFactory.Rbac().V1().ClusterRoles().Lister()
}

func (c *CacheFactory) RoleBindingLister() rbac.RoleBindingLister {
	return c.sharedInformerFactory.Rbac().V1().RoleBindings().Lister()
}

func (c *CacheFactory) ClusterRoleBindingLister() rbac.ClusterRoleBindingLister {
	return c.sharedInformerFactory.Rbac().V1().ClusterRoleBindings().Lister()
}

func (c *CacheFactory) ServiceAccountLister() v1.ServiceAccountLister {
	return c.sharedInformerFactory.Core().V1().ServiceAccounts().Lister()
}

func (c *CacheFactory) EnableBidirectionalSync() {
	c.bidirectionalSync = true
}

func (c *CacheFactory) DisableBidirectionalSync() {
	c.bidirectionalSync = false
}

type eventHandler struct {
	cacheFactory *CacheFactory
	cluster      manager.ICluster
	manager      manager.IK8sResourceManager
}

func newEventHandler(cacheF *CacheFactory, cluster manager.ICluster, man manager.IK8sResourceManager) cache.ResourceEventHandler {
	return &eventHandler{
		cacheFactory: cacheF,
		cluster:      cluster,
		manager:      man,
	}
}

func (h eventHandler) run(f func(ctx context.Context, userCred mcclient.TokenCredential, cls manager.ICluster)) {
	// cacheFactory must enable bidirectional sync
	if !h.cacheFactory.bidirectionalSync {
		return
	}

	adminCred := auth.AdminCredential()
	ctx := context.Background()
	now := time.Now()
	ms := now.UnixMilli()
	ctx = context.WithValue(ctx, "Time", ms)
	lockman.LockClass(ctx, h.manager, db.GetLockClassKey(h.manager, adminCred))
	defer lockman.ReleaseClass(ctx, h.manager, db.GetLockClassKey(h.manager, adminCred))

	// eventWorkMan.Run(func() {
	// f(ctx, adminCred, h.cluster)
	// }, nil, nil)
	f(ctx, adminCred, h.cluster)
}

func (h eventHandler) OnAdd(obj interface{}) {
	h.run(func(ctx context.Context, userCred mcclient.TokenCredential, cls manager.ICluster) {
		h.manager.OnRemoteObjectCreate(ctx, userCred, cls, h.manager, obj.(runtime.Object))
	})
}

func (h eventHandler) OnUpdate(oldObj, newObj interface{}) {
	h.run(func(ctx context.Context, userCred mcclient.TokenCredential, cls manager.ICluster) {
		h.manager.OnRemoteObjectUpdate(ctx, userCred, cls, h.manager, oldObj.(runtime.Object), newObj.(runtime.Object))
	})
}

func (h eventHandler) OnDelete(obj interface{}) {
	h.run(func(ctx context.Context, userCred mcclient.TokenCredential, cls manager.ICluster) {
		h.manager.OnRemoteObjectDelete(ctx, userCred, cls, h.manager, obj.(runtime.Object))
	})
}
