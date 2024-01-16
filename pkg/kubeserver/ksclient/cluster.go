package ksclient

import "C"
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1alpha1 "kubesphere.io/api/cluster/v1alpha1"
	"net/http"
	"net/url"
	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

var baseUrl string = "http://10.19.195.38:30880/apis/cluster.kubesphere.io/v1alpha1/clusters"

func CreateCluster(inpurt api.KsClusterCreateInput) (string, error) {
	kscluster := clusterv1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "cluster.kubesphere.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        inpurt.Name,
			Annotations: map[string]string{"kubesphere.io/creator": "admin"},
		},
		Spec: clusterv1alpha1.ClusterSpec{
			Provider: "",
			Connection: clusterv1alpha1.Connection{
				Type:       "direct",
				KubeConfig: []byte(inpurt.ImportData.Kubeconfig),
			},
			JoinFederation: true,
		},
	}
	clusterJson, err := json.Marshal(kscluster)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	fmt.Println(clusterJson)

	req, err := http.NewRequest("POST", baseUrl, bytes.NewBuffer(clusterJson))
	if err != nil {
		fmt.Println("创建请求失败:", err)
		return "", err
	}

	// 设置请求头，如果需要的话
	req.Header.Set("Content-Type", "application/json")

	// 发送请求并获取响应
	client := GetKsClient()
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("发送请求失败:", err)
		return "", err
	}
	defer resp.Body.Close()
	// 读取响应的内容
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取响应失败:", err)
		return "", err
	}

	// 输出响应内容
	// fmt.Println("响应结果:", string(respBody))
	return string(respBody), nil
}

func ListCluster() (clusterv1alpha1.ClusterList, error) {
	queryParams := url.Values{} // 创建一个空的URL查询参数对象

	// 添加查询参数
	queryParams.Set("limit", "10")
	// queryParams.Set("labelSelector", "cluster-role.kubesphere.io:host")
	queryParams.Set("sortBy", "createTime")

	// 构建完整的URL
	fullURL := baseUrl + "?" + queryParams.Encode()

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		fmt.Println("创建请求失败:", err)
		return clusterv1alpha1.ClusterList{}, err
	}

	// 设置请求头，如果需要的话
	req.Header.Set("Content-Type", "application/json")

	// 发送请求并获取响应
	client := GetKsClient()
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("发送请求失败:", err)
		return clusterv1alpha1.ClusterList{}, err
	}
	defer resp.Body.Close()
	// 读取响应的内容
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取响应失败:", err)
		return clusterv1alpha1.ClusterList{}, err
	}
	var clusterList clusterv1alpha1.ClusterList
	fmt.Println(string(respBody))
	err = json.Unmarshal(respBody, &clusterList)
	if err != nil {
		return clusterv1alpha1.ClusterList{}, err
	}
	for _, kscluster := range clusterList.Items {
		fmt.Println(kscluster.Name)
	}
	return clusterList, nil
}

func DeleteCluster(name string) error {
	// 构建完整的URL
	fullURL := baseUrl + "/" + name

	req, err := http.NewRequest("DELETE", fullURL, nil)
	if err != nil {
		fmt.Println("创建请求失败:", err)
		return err
	}

	// 设置请求头，如果需要的话
	req.Header.Set("Content-Type", "application/json")

	// 发送请求并获取响应
	client := GetKsClient()
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("发送请求失败:", err)
		return err
	}
	defer resp.Body.Close()
	// 读取响应的内容
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取响应失败:", err)
		return err
	}
	var clusterList clusterv1alpha1.ClusterList
	fmt.Println(string(respBody))
	err = json.Unmarshal(respBody, &clusterList)
	if err != nil {
		return err
	}
	for _, kscluster := range clusterList.Items {
		fmt.Println(kscluster.Name)
	}
	return nil
}
