package marathon

import (
	"net"
	"time"
)

type appDockerResponse struct {
	Image string `json:"image"`
}

type appContainerResponse struct {
	Docker appDockerResponse `json:"docker"`
}

type taskHealthCheckResponse struct {
	Alive bool `json:"alive"`
}

type taskResponse struct {
	ID                 string                    `json:"id"`
	Host               net.IP                    `json:"host"`
	State              string                    `json:"state"`
	StagedAt           time.Time                 `json:"stagedAt"`
	StartedAt          time.Time                 `json:"startedAt"`
	HealthCheckResults []taskHealthCheckResponse `json:"healthCheckResults"`
}

type appVersionInfoResponse struct {
	LastConfigChangeAt time.Time `json:"lastConfigChangeAt"`
	LastScalingAt      time.Time `json:"lastScalingAt"`
}

type appResponse struct {
	ID          string                 `json:"id"`
	CPU         float32                `json:"cpus"`
	GPU         float32                `json:"gpus"`
	Mem         uint                   `json:"mem"`
	Disk        uint                   `json:"disk"`
	Labels      map[string]string      `json:"labels"`
	Env         map[string]string      `json:"env"`
	Constraints [][]string             `json:"constraints"`
	Container   appContainerResponse   `json:"container"`
	Tasks       []taskResponse         `json:"tasks"`
	Version     time.Time              `json:"version"`
	VersionInfo appVersionInfoResponse `json:"versionInfo"`
}

type appsResponse struct {
	Apps []appResponse `json:"apps"`
}
