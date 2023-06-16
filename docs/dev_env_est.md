# Development environment establishment环境搭建

## Registry仓库

### 1. 搭建
1. 可以通过[代码](https://github.com/distribution/distribution)，[可执行文件](https://github.com/distribution/distribution/releases/tag/v2.8.2)，[镜像容器](https://docs.docker.com/registry/deploying/)搭建。   
2. 运行前先修改配置，其中包括[Auth](https://docs.docker.com/registry/deploying/)和[TLS](https://docs.docker.com/registry/deploying/)以及其他配置。
3. 使用[tls非安全的方案](https://docs.docker.com/registry/insecure/#deploy-a-plain-http-registry)是不推荐的,如果要用需要去改*docker engine*的*daemon.json*配置文件。
```json
{
  "insecure-registries" : ["http://myregistrydomain.com:5000"]
}
```
4. 如果没有CA给你发证书，也可以通过[自签证书](https://docs.docker.com/registry/insecure/#use-self-signed-certificates)来运行。
```shell
mkdir -p certs
openssl req \
  -newkey rsa:4096 -nodes -sha256 -keyout certs/domain.key \
  -addext "subjectAltName = DNS:registry.luoruofeng.com" \
  -x509 -days 365 -out certs/domain.crt
```


### 2. 使用
#### 登录
* [registry使用命令行登录](https://linuxhint.com/pull-docker-image-from-private-registry/)
登录后可以查看*cat ~/.docker/config.json*
```json
{
    "auths": {
        "https://index.docker.io/v1/": {
            "auth": "c3R...zE2"
        }
    }
}
```

#### 推送
* 可以使用[docker](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/)将image推送到registry。推送前需要先登录。     
* 如果需要[查询registry上的镜像信息](https://linuxhint.com/pull-docker-image-from-private-registry/)请使用registry提供的API 
* 当然所有使用registry的API的方法请以[官网](https://docs.docker.com/registry/spec/api/)为准。

#### 拉取
* 最基本的拉取操作当然是从[docker](https://linuxhint.com/pull-docker-image-from-private-registry/)开始。
* 当然如果你使用的容器编排工具是[k8s](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/)你需要更多的配置操作。

<br><br>

## minikube搭建
* 可以适配多种环境进行[安装](https://minikube.sigs.k8s.io/docs/start/)。具体操作教程参考[官网教程](https://minikube.sigs.k8s.io/docs/tutorials/)为准。   
* minikube使用没auth和tls的registry的[教程](https://gist.github.com/trisberg/37c97b6cc53def9a3e38be6143786589)。