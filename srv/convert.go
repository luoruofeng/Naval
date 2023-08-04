package srv

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kubernetes/kompose/pkg/kobject"
	"github.com/luoruofeng/Naval/model"
	"go.uber.org/zap"
)

var (
	ConvertOut            string
	ConvertBuildRepo      string
	ConvertBuildBranch    string
	ConvertBuild          string
	ConvertVolumes        string
	ConvertPVCRequestSize string
	ConvertChart          bool

	ConvertYaml              bool
	ConvertJSON              bool
	ConvertStdout            bool
	ConvertEmptyVols         bool
	ConvertInsecureRepo      bool
	ConvertDeploymentConfig  bool
	ConvertReplicas          int
	ConvertController        string
	ConvertPushImage         bool
	ConvertPushImageRegistry string
	ConvertOpt               kobject.ConvertOptions
	ConvertYAMLIndent        int
	GenerateNetworkPolicies  bool

	UpBuild string

	BuildCommand string
	PushCommand  string
	// WithKomposeAnnotation decides if we will add metadata about this convert to resource's annotation.
	// default is true.
	WithKomposeAnnotation bool

	// MultipleContainerMode which enables creating multi containers in a single pod is a developping function.
	// default is false
	MultipleContainerMode bool

	ServiceGroupMode string
	ServiceGroupName string

	// SecretsAsFiles forces secrets to result in files inside a container instead of symlinked directories containing
	// files of the same name. This reproduces the behavior of file-based secrets in docker-compose and should probably
	// be the default for kompose, but we must keep compatibility with the previous behavior.
	// See https://github.com/kubernetes/kompose/issues/1280 for more details.
	SecretsAsFiles bool
)

func init() {
	// Kubernetes only
	ConvertChart = false             //Create a Helm chart for converted objects
	ConvertController = "deployment" // `Set the output controller ("deployment"|"daemonSet"|"replicationController")`)

	MultipleContainerMode = false //Create multiple containers grouped by 'kompose.service.group' label")
	ServiceGroupMode = ""         //Group multiple service to create single workload by `label`(`kompose.service.group`) or `volume`(shared volumes)")
	ServiceGroupName = ""         //Using with --service-group-mode=volume to specific a final service name for the group")
	SecretsAsFiles = false        //Always convert docker-compose secrets into files instead of symlinked directories.")

	// OpenShift only
	ConvertDeploymentConfig = true //Generate an OpenShift deploymentconfig object")
	ConvertInsecureRepo = false    //Use an insecure Docker repository for OpenShift ImageStream")
	ConvertBuildRepo = ""          //Specify source repository for buildconfig (default remote origin)")
	ConvertBuildBranch = ""        //Specify repository branch to use for buildconfig (default master)")

	// Standard between the two
	ConvertBuild = "none"                    //Set the type of build ("local"|"build-config"(OpenShift only)|"none")`)
	ConvertPushImage = false                 //If we should push the docker image we built")
	BuildCommand = ""                        //Set the command used to build the container image. override the docker build command.Should be used in conjuction with --push-command flag.`)
	PushCommand = ""                         //Set the command used to push the container image. override the docker push command. Should be used in conjuction with --build-command flag.`)
	ConvertPushImageRegistry = ""            //Specify registry for pushing image, which will override registry from image name.")
	ConvertYaml = false                      //Generate resource files into YAML format")
	ConvertJSON = false                      //Generate resource files into JSON format")
	ConvertStdout = false                    //Print converted objects to stdout")
	ConvertOut = ""                          //Specify a file name or directory to save objects to (if path does not exist, a file will be created)")
	ConvertReplicas = 1                      //Specify the number of replicas in the generated resource spec")
	ConvertVolumes = "persistentVolumeClaim" //Volumes to be generated ("persistentVolumeClaim"|"emptyDir"|"hostPath" | "configMap")`)
	ConvertPVCRequestSize = ""               //Specify the size of pvc storage requests in the generated resource spec`)
	GenerateNetworkPolicies = false          //Specify whether to generate network policies or not.")

	WithKomposeAnnotation = true //Add kompose annotations to generated resource")

	// Deprecated commands
	ConvertEmptyVols = false //Use Empty Volumes. Do not generate PVCs")
	ConvertYAMLIndent = 2    //Spaces length to indent generated yaml files")
}

func PreRun(controllerType *model.ControllerType, replicas *int, inputFiles []string, outFile string) kobject.ConvertOptions {
	var (
		controllerValue         string = ConvertController
		replicasValue           int    = ConvertReplicas
		isChangeReplicaSetValue        = false
	)

	if replicas != nil {
		replicasValue = *replicas
		isChangeReplicaSetValue = true
	}

	if controllerType != nil {
		//deployment"|"daemonSet"|"replicationController
		if *controllerType == model.ReplicationController {
			controllerValue = "replicationController"
		} else if *controllerType == model.DaemonSet {
			controllerValue = "daemonSet"
		} else if *controllerType == model.Statefulset {
			controllerValue = "statefulset"
		}
	}

	// Create the Convert Options.
	ConvertOpt = kobject.ConvertOptions{
		ToStdout:                    ConvertStdout,
		CreateChart:                 ConvertChart,
		GenerateYaml:                ConvertYaml,
		GenerateJSON:                ConvertJSON,
		Replicas:                    replicasValue,
		InputFiles:                  inputFiles,
		OutFile:                     outFile,
		Provider:                    "kubernetes",
		Build:                       ConvertBuild,
		BuildRepo:                   ConvertBuildRepo,
		BuildBranch:                 ConvertBuildBranch,
		PushImage:                   ConvertPushImage,
		PushImageRegistry:           ConvertPushImageRegistry,
		CreateDeploymentConfig:      ConvertDeploymentConfig,
		EmptyVols:                   ConvertEmptyVols,
		Volumes:                     ConvertVolumes,
		PVCRequestSize:              ConvertPVCRequestSize,
		InsecureRepository:          ConvertInsecureRepo,
		IsDaemonSetFlag:             false,
		IsReplicationControllerFlag: false,
		Controller:                  strings.ToLower(controllerValue),
		IsReplicaSetFlag:            isChangeReplicaSetValue,
		YAMLIndent:                  ConvertYAMLIndent,
		WithKomposeAnnotation:       WithKomposeAnnotation,
		MultipleContainerMode:       MultipleContainerMode,
		ServiceGroupMode:            ServiceGroupMode,
		ServiceGroupName:            ServiceGroupName,
		SecretsAsFiles:              SecretsAsFiles,
		GenerateNetworkPolicies:     GenerateNetworkPolicies,
		BuildCommand:                BuildCommand,
		PushCommand:                 PushCommand,
	}

	if ServiceGroupMode == "" && MultipleContainerMode {
		ConvertOpt.ServiceGroupMode = "label"
	}
	return ConvertOpt
}

func CreateDockerComposeFile(log *zap.Logger, i int, dockerComposeContent string, folderName string) (*string, error) {
	log.Info("创建DockerCompose文件", zap.Int("convert item index", i), zap.Any("dockerComposeContent", dockerComposeContent))
	filePath := filepath.Join(folderName, fmt.Sprintf("docker-compose-%d.yml", i))
	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	_, err = file.WriteString(dockerComposeContent)
	if err != nil {
		return nil, err
	}

	return &filePath, nil
}

// if folder is not exist, create it
func CreateFolder(log *zap.Logger, folder string) error {
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		log.Info("转换任务-创建文件夹", zap.String("folder", folder))
		err := os.Mkdir(folder, 0777)
		if err != nil {
			return err
		}
	}
	return nil
}

// Convert docker-compose.yml to k8s yaml
func (ts *TaskSrv) Convert(task *model.Task) error {
	log := ts.logger
	tmpFolder := ts.cnf.SaveComposeTmpFolder
	needDeleteConvertFolder := ts.cnf.NeedDeleteConvertFolder
	mongoSrv := ts.mongoT
	needExecuteImmediately := ts.cnf.NeedExecuteImmediately

	log.Info("转换任务-开始", zap.String("task.id", task.Id))
	// 创建总文件夹
	err := CreateFolder(log, tmpFolder)
	if err != nil {
		log.Error("转换任务-创建总文件夹失败", zap.String("tmpFolder", tmpFolder), zap.Error(err))
		return err
	}
	folderName := filepath.Join(tmpFolder, fmt.Sprintf("%s-%s", task.Id, time.Now().Format("20060102150405")))

	// 创建子文件夹-用于存放yml的文件夹
	err = CreateFolder(log, folderName)
	if err != nil {
		log.Error("转换任务-创建子文件夹-用于存放yml的文件夹失败", zap.String("folderName", folderName), zap.Error(err))
		return err
	}

	defer func() {
		// 删除子文件夹
		if needDeleteConvertFolder {
			err := os.RemoveAll(folderName)
			if err != nil {
				log.Error("转换任务-删除临时文件夹用于存储k8s的yaml文件失败", zap.String("folderName", folderName), zap.Error(err))
			} else {
				log.Info("转换任务-删除临时文件夹用于存储k8s的yaml文件", zap.String("folderName", folderName))
			}
		}
	}()

	items := make([]model.Item, 0)
	for i, item := range task.Kompose.Items {
		// 创建docker-compose.yml文件
		composeFilePath, err := CreateDockerComposeFile(log, i, item.DockerComposeContent, folderName)
		if err != nil {
			log.Error("转换任务-创建docker-compose.yml文件失败", zap.Int("convert item index", i), zap.Error(err))
			return err
		}
		log.Info("转换任务-创建docker-compose.yml文件成功", zap.Int("convert item index", i), zap.String("composeFilePath", *composeFilePath))

		// 转换docker-compose.yml文件
		convertOptions := PreRun(item.ControllerType, item.Replicas, []string{*composeFilePath}, folderName)
		previousFiles, err := ReadDirAllFiles(folderName) // get previous files
		if err != nil {
			log.Error("转换任务-读取文件夹失败", zap.String("folderName", folderName), zap.Error(err))
			return err
		}
		log.Info("转换任务-转换中", zap.Any("convertOptions", convertOptions))
		// 本来应该直接调用app.Convert(convertOptions)但是这个方法会调用log.Fatalf退出程序，所以这里直接调用自己封装的SrvConvert。
		err = SrvConvert(convertOptions, log)
		if err != nil {
			log.Error("转换任务-转换失败", zap.Any("convertOptions", convertOptions), zap.Error(err))
			return err
		}

		// 获取转换后的文件
		currentFiles, err := ReadDirAllFiles(folderName)
		if err != nil {
			return err
		}
		newFiles := CompareFiles(previousFiles, currentFiles)

		// 读取转换后新生成的文件的内容，保存到mongoDB
		for _, newFile := range newFiles {
			fileContent, err := os.ReadFile(newFile)
			if err != nil {
				log.Error("转换任务-读取转换后新生成的文件的内容失败", zap.String("newFile", newFile), zap.Error(err))
				return err
			}
			log.Info("转换任务-读取转换后新生成的文件的内容成功", zap.String("newFile", newFile), zap.String("fileContent", string(fileContent)))
			item := model.Item{
				FilePath:       newFile,
				K8SYamlContent: string(fileContent),
			}
			items = append(items, item)
		}
	}
	log.Info("转换任务-成功。mongoDB中更新字段:Items:items Type:create ConvertSuccessfully:true StateCode:Pendding", zap.Any("Items", items))
	mongoSrv.UpdateKVs(task.MongoId, map[string]interface{}{"Items": items, "Type": model.Create, "ConvertSuccessfully": true, "StateCode": model.Pending, "IsRunning": false})

	if needExecuteImmediately {
		//TODO 执行任务
		log.Info("转换任务-结束。立即开始执行任务。")
		ts.Update(*task)
	} else {
		log.Info("转换任务-结束。")
	}

	return nil
}

// 对比previousFiles和currentFiles，找到previousFiles中不存在的文件，即为新生成的文件
func CompareFiles(previousFiles []string, currentFiles []string) []string {
	newFiles := make([]string, 0)
	for _, currentFile := range currentFiles {
		isExist := false
		for _, previousFile := range previousFiles {
			if currentFile == previousFile {
				isExist = true
				break
			}
		}
		if !isExist {
			newFiles = append(newFiles, currentFile)
		}
	}
	return newFiles
}

// read dir all files and reture file name list
func ReadDirAllFiles(dir string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		files = append(files, path)
		return nil
	})
	return files, err
}
