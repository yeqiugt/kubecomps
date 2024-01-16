package ksclient

import (
	"context"
	"fmt"
	"golang.org/x/oauth2"
	"io/ioutil"
	"net/http"
)

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
