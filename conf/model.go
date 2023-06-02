package conf

type Config struct {
	LogLevel             string `json:"log_level"`
	LogFile              string `json:"log_file"`
	HttpAddr             string `json:"http_addr"`
	HttpReadOverTime     int    `json:"http_read_over_time"`
	HttpWriteOverTime    int    `json:"http_write_over_time"`
	RegistryAddr         string `json:"registry_addr"`
	DockerApiServer      string `json:"docker_api_server"`
	DockerSwarmApiServer string `json:"docker_swarm_api_server"`
	K8sApiServer         string `json:"k8s_api_server"`
}
