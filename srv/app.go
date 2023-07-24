package srv

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"os"

	"github.com/kubernetes/kompose/pkg/kobject"
	"github.com/kubernetes/kompose/pkg/loader"
	"github.com/kubernetes/kompose/pkg/transformer"
	"github.com/kubernetes/kompose/pkg/transformer/kubernetes"
	"github.com/kubernetes/kompose/pkg/transformer/openshift"
)

var (
	log *zap.Logger
	// DefaultComposeFiles is a list of filenames that kompose will use if no file is explicitly set
	DefaultComposeFiles = []string{
		"compose.yaml",
		"compose.yml",
		"docker-compose.yaml",
		"docker-compose.yml",
	}
)

const (
	// ProviderKubernetes is provider kubernetes
	ProviderKubernetes = "kubernetes"
	// ProviderOpenshift is provider openshift
	ProviderOpenshift = "openshift"
	// DefaultProvider - provider that will be used if there is no provider was explicitly set
	DefaultProvider = ProviderKubernetes
)

var inputFormat = "compose"

// ValidateFlags validates all command line flags
func ValidateFlags(args []string, cmd *cobra.Command, opt *kobject.ConvertOptions) {
	if opt.OutFile == "-" {
		opt.ToStdout = true
		opt.OutFile = ""
	}

	// Get the provider
	provider := cmd.Flags().Lookup("provider").Value.String()
	log.Debug(fmt.Sprintf("Checking validation of provider: %s", provider))

	// OpenShift specific flags
	deploymentConfig := cmd.Flags().Lookup("deployment-config").Changed
	buildRepo := cmd.Flags().Lookup("build-repo").Changed
	buildBranch := cmd.Flags().Lookup("build-branch").Changed

	// Kubernetes specific flags
	chart := cmd.Flags().Lookup("chart").Changed
	daemonSet := cmd.Flags().Lookup("daemon-set").Changed
	replicationController := cmd.Flags().Lookup("replication-controller").Changed
	deployment := cmd.Flags().Lookup("deployment").Changed

	// Get the controller
	controller := opt.Controller
	log.Debug(fmt.Sprintf("Checking validation of controller: %s", controller))

	// Check validations against provider flags
	switch {
	case provider == ProviderOpenshift:
		if chart {
			log.Error("--chart, -c is a Kubernetes only flag")
			return
		}
		if daemonSet {
			log.Error("--daemon-set is a Kubernetes only flag")
			return
		}
		if replicationController {
			log.Error("--replication-controller is a Kubernetes only flag")
			return
		}
		if deployment {
			log.Error("--deployment, -d is a Kubernetes only flag")
		}
		if controller == "daemonset" || controller == "replicationcontroller" || controller == "deployment" {
			log.Error("--controller= daemonset, replicationcontroller or deployment is a Kubernetes only flag")
			return
		}
	case provider == ProviderKubernetes:
		if deploymentConfig {
			log.Error("--deployment-config is an OpenShift only flag")
			return
		}
		if buildRepo {
			log.Error("--build-repo is an Openshift only flag")
			return
		}
		if buildBranch {
			log.Error("--build-branch is an Openshift only flag")
			return
		}
		if controller == "deploymentconfig" {
			log.Error("--controller=deploymentConfig is an OpenShift only flag")
			return
		}
	}

	// Standard checks regardless of provider
	if len(opt.OutFile) != 0 && opt.ToStdout {
		log.Error("Error: --out and --stdout can't be set at the same time")
		return
	}

	if opt.CreateChart && opt.ToStdout {
		log.Error("Error: chart cannot be generated when --stdout is specified")
		return
	}

	if opt.Replicas < 0 {
		log.Error("Error: --replicas cannot be negative")
		return
	}

	if len(args) != 0 {
		log.Error("Unknown Argument(s)", zap.String("args", strings.Join(args, ",")))
		return
	}

	if opt.GenerateJSON && opt.GenerateYaml {
		log.Error("YAML and JSON format cannot be provided at the same time")
		return
	}

	if _, ok := kubernetes.ValidVolumeSet[opt.Volumes]; !ok {
		validVolumesTypes := make([]string, 0)
		for validVolumeType := range kubernetes.ValidVolumeSet {
			validVolumesTypes = append(validVolumesTypes, fmt.Sprintf("'%s'", validVolumeType))
		}
		log.Error(fmt.Sprintf("Unknown Volume type: ", opt.Volumes, ", possible values are: ", strings.Join(validVolumesTypes, " ")))
		return
	}
}

// ValidateComposeFile validates the compose file provided for conversion
func ValidateComposeFile(opt *kobject.ConvertOptions) {
	if len(opt.InputFiles) == 0 {
		for _, name := range DefaultComposeFiles {
			_, err := os.Stat(name)
			if err != nil {
				log.Debug(fmt.Sprintf("'%s' not found: %v", name, err))
			} else {
				opt.InputFiles = []string{name}
				return
			}
		}

		log.Error("No 'docker-compose' file found")
		return
	}
}

func validateControllers(opt *kobject.ConvertOptions) {
	singleOutput := len(opt.OutFile) != 0 || opt.OutFile == "-" || opt.ToStdout
	if opt.Provider == ProviderKubernetes {
		// create deployment by default if no controller has been set
		if !opt.CreateD && !opt.CreateDS && !opt.CreateRC && opt.Controller == "" {
			opt.CreateD = true
		}
		if singleOutput {
			count := 0
			if opt.CreateD {
				count++
			}
			if opt.CreateDS {
				count++
			}
			if opt.CreateRC {
				count++
			}
			if count > 1 {
				log.Error("Error: only one kind of Kubernetes resource can be generated when --out or --stdout is specified")
				return
			}
		}
	} else if opt.Provider == ProviderOpenshift {
		// create deploymentconfig by default if no controller has been set
		if !opt.CreateDeploymentConfig {
			opt.CreateDeploymentConfig = true
		}
		if singleOutput {
			count := 0
			if opt.CreateDeploymentConfig {
				count++
			}
			// Add more controllers here once they are available in OpenShift
			// if opt.foo {count++}

			if count > 1 {
				log.Error("Error: only one kind of OpenShift resource can be generated when --out or --stdout is specified")
				return
			}
		}
	}
}

// Convert transforms docker compose or dab file to k8s objects
func SrvConvert(opt kobject.ConvertOptions, log *zap.Logger) error {
	log = log
	validateControllers(&opt)

	// loader parses input from file into komposeObject.
	l, err := loader.GetLoader(inputFormat)
	if err != nil {
		log.Error("err:", zap.Error(err))
		return err
	}

	komposeObject := kobject.KomposeObject{
		ServiceConfigs: make(map[string]kobject.ServiceConfig),
	}
	komposeObject, err = l.LoadFile(opt.InputFiles)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	komposeObject.Namespace = opt.Namespace

	// Get a transformer that maps komposeObject to provider's primitives
	t := getTransformer(opt)

	// Do the transformation
	objects, err := t.Transform(komposeObject, opt)

	if err != nil {
		log.Error(err.Error())
		return err
	}

	// Print output
	err = kubernetes.PrintList(objects, opt)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}

// Convenience method to return the appropriate Transformer based on
// what provider we are using.
func getTransformer(opt kobject.ConvertOptions) transformer.Transformer {
	var t transformer.Transformer
	if opt.Provider == DefaultProvider {
		// Create/Init new Kubernetes object with CLI opts
		t = &kubernetes.Kubernetes{Opt: opt}
	} else {
		// Create/Init new OpenShift object that is initialized with a newly
		// created Kubernetes object. Openshift inherits from Kubernetes
		t = &openshift.OpenShift{Kubernetes: kubernetes.Kubernetes{Opt: opt}}
	}
	return t
}
