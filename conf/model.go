package conf

type Config struct {
	LogLevel             string `json:"log_level"`
	LogFile              string `json:"log_file"`
	HttpAddr             string `json:"http_addr"`
	HttpReadOverTime     int    `json:"http_read_over_time"`
	HttpWriteOverTime    int    `json:"http_write_over_time"`
	RegistryAddr         string `json:"registry_addr"`
	RegistryEmbedded     bool   `json:"registry_embedded"`
	RegistryEnableAuth   bool   `json:"registry_enable_auth"`
	RegistryAuthUser     string `json:"registry_auth_user"`
	RegistryAuthPassword string `json:"registry_auth_password"`
	RegistryEnableTls    bool   `json:"registry_enable_tls"`
	RegistryTlsKey       string `json:"registry_tls_key"`
	RegistryTlsCert      string `json:"registry_tls_cert"`
	RegistryDataDir      string `json:"registry_data_dir"`
	RegistryLogDir       string `json:"registry_log_dir"`
	RegistryHtpasswdPath string `json:"registry_htpasswd_path"`
	RegistryLogLevel     string `json:"registry_log_level"`
	DockerApiServer      string `json:"docker_api_server"`
	DockerSwarmApiServer string `json:"docker_swarm_api_server"`
	K8sApiServer         string `json:"k8s_api_server"`
}
