package kube

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/luoruofeng/Naval/util"
	"go.uber.org/fx"
	"go.uber.org/zap"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"
)

type TaskKubeSrv struct {
	logger *zap.Logger
	dc     *dynamic.DynamicClient
	// deploymentRes           schema.GroupVersionResource
	// serviceRes              schema.GroupVersionResource
	groupVersionResourceMap map[string]schema.GroupVersionResource
}

func NewGroupVersionResourceMap() map[string]schema.GroupVersionResource {
	result := make(map[string]schema.GroupVersionResource, 0)
	deploymentRes := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	serviceRes := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	podRes := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	persistentVolumeRes := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumes"}
	persistentVolumeClaimRes := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumeclaims"}

	result["Deployment"] = deploymentRes
	result["Service"] = serviceRes
	result["Pod"] = podRes
	result["PersistentVolume"] = persistentVolumeRes
	result["PersistentVolumeClaim"] = persistentVolumeClaimRes
	return result
}

func NewTaskKubeSrv(lc fx.Lifecycle, logger *zap.Logger, ctx context.Context) *TaskKubeSrv {
	log := logger
	result := TaskKubeSrv{
		logger:                  logger,
		groupVersionResourceMap: NewGroupVersionResourceMap(),
	}
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			log.Info("初始化TaskKubeSrv服务")
			var kubeconfig string
			if home := homedir.HomeDir(); home != "" {
				kubeconfig = filepath.Join(home, ".kube", "config")
			} else {
				// 获取当前工作目录
				cwd, err := os.Getwd()
				if err != nil {
					log.Error("启动TaskKube服务-获取当前工作目录失败", zap.Error(err))
					return err
				} else {
					log.Info("启动TaskKube服务-当前目录", zap.String("path", cwd))
				}
				rootDir := util.GetRootProjPath(cwd)
				kubeconfig = filepath.Join(rootDir, "config", "kube-config.yml")
			}
			log.Info("启动TaskKube服务-读取kube配置文件", zap.String("path", kubeconfig))
			config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				log.Error("启动TaskKube服务-构建配置文件失败 BuildConfigFromFlags", zap.Error(err))
				return err
			}
			client, err := dynamic.NewForConfig(config)
			if err != nil {
				log.Error("启动TaskKube服务-构建配置文件失败 NewForConfig", zap.Error(err))
				return err
			}
			result.dc = client
			return nil
		}, OnStop: func(context.Context) error {
			log.Info("停止TaskKubeSrv服务")
			return nil
		},
	})
	return &result
}

// 创建Deployment
func (tks TaskKubeSrv) Create(resouceYml string) error {
	log := tks.logger
	var resouceKind string
	resource, err := YmlToUnstructured(resouceYml)
	if err != nil {
		log.Error("创建resouceYml-转换yaml格式失败", zap.String("resouceYml", resouceYml), zap.Error(err))
		return err
	}
	resouceKind = resource.GetKind()
	switch resouceKind {
	case "Deployment", "Service", "Pod", "PersistentVolume", "PersistentVolumeClaim":
		// Create resource
		log.Info("创建resouceYml-转换yaml格式", zap.String("resouceYml", resouceYml))
		groupVersionResource := tks.groupVersionResourceMap[resouceKind]
		if resource, err := YmlToUnstructured(resouceYml); err != nil {
			log.Error("创建resouceYml-转换yaml格式失败", zap.String("resouceYml", resouceYml), zap.Error(err))
			return err
		} else {
			log.Info("创建"+resouceKind+"-开始", zap.Any(resouceKind+" obj", resource.Object), zap.Any(resouceKind+"Res", groupVersionResource))
			result, err := tks.dc.Resource(groupVersionResource).Namespace(apiv1.NamespaceDefault).Create(context.TODO(), resource, metav1.CreateOptions{})
			if err != nil {
				log.Error("创建"+resouceKind+"-失败", zap.Any(resouceKind, resource), zap.Error(err))
				return err
			}
			log.Info("创建"+resouceKind+"-成功", zap.String("name", result.GetName()), zap.String("resouceYml", resouceYml))
		}
	default:
		log.Error("创建resouceYml-不支持的资源类型", zap.String("resouceYml", resouceYml), zap.Any("resource", resource))
	}
	return nil
}

// 删除Deployment
func (tks *TaskKubeSrv) Delete(resouceKind string, resouceName string) error {
	log := tks.logger

	switch resouceKind {
	case "Deployment", "Service", "Pod", "PersistentVolume", "PersistentVolumeClaim":
		groupVersionResource := tks.groupVersionResourceMap[resouceKind]
		deletePolicy := metav1.DeletePropagationForeground
		deleteOptions := metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		}
		if err := tks.dc.Resource(groupVersionResource).Namespace(apiv1.NamespaceDefault).Delete(context.TODO(), resouceName, deleteOptions); err != nil {
			log.Error("删除"+resouceKind+"-失败", zap.Error(err))
			return err
		}
		log.Info("删除"+resouceKind+"-成功", zap.String(resouceKind+" name", resouceName))
	default:
		log.Error("删除resouceYml-不支持的资源类型", zap.String("resouceKind", resouceKind), zap.String("resouceName", resouceName))
		return fmt.Errorf("删除resouceYml-不支持的资源类型 resouceKind: %s, resouceName: %s", resouceKind, resouceName)
	}
	return nil
}

func (tks *TaskKubeSrv) UpdateDeployReplicasNumber(deploymentName string, n int) (bool, error) {
	log := tks.logger
	deploymentRes := tks.groupVersionResourceMap["Deployment"]
	log.Info("更新deployment的replicas数量-开始", zap.String("deployment name", deploymentName), zap.Int("replicas number", n))
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := tks.dc.Resource(deploymentRes).Namespace(apiv1.NamespaceDefault).Get(context.TODO(), deploymentName, metav1.GetOptions{})
		if getErr != nil {
			log.Error("更新deployment的replicas数量-通过deployment名字获取出错", zap.String("deployment name", deploymentName), zap.Error(getErr))
			return getErr
		}

		if err := unstructured.SetNestedField(result.Object, int64(n), "spec", "replicas"); err != nil {
			log.Error("更新deployment的replicas数量-设置更新信息错误", zap.String("deployment name", deploymentName), zap.Any("obj", result.Object), zap.Int("replicas number", n), zap.Error(err))
			return err
		}
		_, updateErr := tks.dc.Resource(deploymentRes).Namespace(apiv1.NamespaceDefault).Update(context.TODO(), result, metav1.UpdateOptions{})
		if updateErr != nil {
			log.Error("更新deployment的replicas数量-更新失败", zap.String("deployment name", deploymentName), zap.Int("replicas number", n), zap.Error(updateErr))
			return updateErr
		}
		return updateErr
	})
	if retryErr != nil {
		log.Error("更新deployment的replicas数量-更新错误", zap.String("deployment name", deploymentName), zap.Int("replicas number", n), zap.Error(retryErr))
		return false, retryErr
	}

	log.Info("更新deployment的replicas数量-成功", zap.String("deployment name", deploymentName), zap.Int("replicas number", n))
	return true, nil
}

func (tks *TaskKubeSrv) UpdateDeployImages(deploymentName string, images ...string) (bool, error) {
	log := tks.logger
	deploymentRes := tks.groupVersionResourceMap["Deployment"]
	log.Info("更新deployment的镜像-开始", zap.String("deployment name", deploymentName), zap.Any("images", images))
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := tks.dc.Resource(deploymentRes).Namespace(apiv1.NamespaceDefault).Get(context.TODO(), deploymentName, metav1.GetOptions{})
		if getErr != nil {
			log.Error("更新deployment的镜像-通过deployment名字获取失败", zap.String("deployment name", deploymentName), zap.Error(getErr))
			return getErr
		}

		containers, found, err := unstructured.NestedSlice(result.Object, "spec", "template", "spec", "containers")
		if err != nil || !found || containers == nil {
			log.Error("更新deployment的镜像-提取容器信息失败", zap.String("deployment name", deploymentName), zap.Any("new images", images), zap.Error(err))
			return err
		}

		if len(containers) != len(images) {
			log.Error("更新deployment的镜像-传入的参数images的数量和已有的容器数量不一致", zap.String("deployment name", deploymentName), zap.Any("new images", images), zap.Error(errors.New("number of images parameter is not equal number of container")))
			return err
		}

		// update container[0] image
		for i, image := range images {
			if err := unstructured.SetNestedField(containers[i].(map[string]interface{}), image, "image"); err != nil {
				log.Error("更新deployment的镜像-设置容器失败", zap.Any("original container", containers[i]), zap.String("deployment name", deploymentName), zap.Any("new image", image), zap.Error(err))
				return err
			}
		}
		if err := unstructured.SetNestedField(result.Object, containers, "spec", "template", "spec", "containers"); err != nil {
			log.Error("更新deployment的镜像-修改容器失败", zap.Any("new containers", containers), zap.String("deployment name", deploymentName), zap.Error(err))
			return err
		}

		_, updateErr := tks.dc.Resource(deploymentRes).Namespace(apiv1.NamespaceDefault).Update(context.TODO(), result, metav1.UpdateOptions{})
		if updateErr != nil {
			log.Error("更新deployment的镜像-更新出错", zap.String("deployment name", deploymentName), zap.Any("images", images), zap.Error(err))
			return err
		}
		return updateErr
	})
	if retryErr != nil {
		log.Error("更新deployment的镜像-更新错误", zap.String("deployment name", deploymentName), zap.Any("images", images), zap.Error(retryErr))
		return false, retryErr
	}

	log.Info("更新deployment的镜像-成功", zap.String("deployment name", deploymentName), zap.Any("image", images))
	return true, nil
}
