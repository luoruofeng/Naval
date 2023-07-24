package conf

type Config struct {
	LogLevel                string `json:"log_level"`
	LogFile                 string `json:"log_file"`
	HttpAddr                string `json:"http_addr"`
	HttpReadOverTime        int    `json:"http_read_over_time"`
	HttpWriteOverTime       int    `json:"http_write_over_time"`
	RegistryAddr            string `json:"registry_addr"`
	DockerApiServer         string `json:"docker_api_server"`
	DockerSwarmApiServer    string `json:"docker_swarm_api_server"`
	K8sApiServer            string `json:"k8s_api_server"`
	SaveComposeTmpFolder    string `json:"save_compose_tmp_folder"`
	NeedDeleteConvertFolder bool   `json:"need_delete_convert_folder"`
	NeedExecuteImmediately  bool   `json:"need_execute_immediately"`
	AsyncConvert            bool   `json:"async_convert"`
}
