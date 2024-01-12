package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1alpha1 "kubesphere.io/api/cluster/v1alpha1"
	"net/http"
	"yunion.io/x/jsonutils"
	"yunion.io/x/kubecomps/pkg/kubeserver/api"
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
	res := api.KsClusterCreateInput{}
	if err := t.GetParams().Unmarshal(&res); err != nil {
		return
	}

	kscluster := clusterv1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "cluster.kubesphere.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        res.Name,
			Annotations: map[string]string{"kubesphere.io/creator": "admin"},
		},
		Spec: clusterv1alpha1.ClusterSpec{
			Provider: "",
			Connection: clusterv1alpha1.Connection{
				Type:       "direct",
				KubeConfig: []byte(res.ImportData.Kubeconfig),
			},
			JoinFederation: true,
		},
	}
	clusterJson, err := json.Marshal(kscluster)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(clusterJson)

	url := "http://10.19.195.38:30880/apis/cluster.kubesphere.io/v1alpha1/clusters"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(clusterJson))
	if err != nil {
		fmt.Println("创建请求失败:", err)
		return
	}

	// 设置请求头，如果需要的话
	req.Header.Set("Content-Type", "application/json")

	// 发送请求并获取响应
	client := GetKsClient()
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("发送请求失败:", err)
		return
	}
	defer resp.Body.Close()
	// 读取响应的内容
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取响应失败:", err)
		return
	}

	// 输出响应内容
	fmt.Println("响应结果:", string(respBody))

}

func GetKsClient() *http.Client {
	ctx := context.Background()
	conf := &oauth2.Config{
		ClientID:     "kubesphere",
		ClientSecret: "kubesphere",
		Endpoint: oauth2.Endpoint{
			// AuthURL:  "https://provider.com/o/oauth2/auth",
			TokenURL: "http://10.19.36.8:30001/oauth/token",
		},
	}

	// Use the authorization code that is pushed to the redirect
	// URL. Exchange will do the handshake to retrieve the
	// initial access token. The HTTP Client returned by
	// conf.Client will refresh the token as necessary.
	// 获取用户的用户名和密码
	username := "admin"
	password := "xdjr0lxGu"

	// 使用密码凭证从 OAuth 提供商获取令牌
	token, err := conf.PasswordCredentialsToken(context.Background(), username, password)
	if err != nil {
		fmt.Printf("无法获取令牌：%v\n", err)
		return nil
	}
	// token.Valid()
	source := conf.TokenSource(ctx, token)
	refreshToken, err := source.Token()
	if err != nil {
		fmt.Println(err)
	}
	// fmt.Println(refreshToken)
	//tok, err := conf.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	//if err != nil {
	//	log.Fatal(err)
	//}

	client := conf.Client(ctx, refreshToken)
	resp, err := client.Get("http://10.19.195.38:30880/apis/cluster.kubesphere.io/v1alpha1/clusters")
	if err != nil {
		fmt.Println(err)
	}
	//fmt.Println(res.Body)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应失败：%v\n", err)
		return nil
	}
	fmt.Println(string(body))
	return client
}
