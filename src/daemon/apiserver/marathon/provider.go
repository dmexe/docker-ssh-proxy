package marathon

import (
	"daemon/apiserver"
	"daemon/payloads"
	"daemon/utils"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// From https://github.com/apache/mesos/blob/master/include/mesos/mesos.proto#L1674
var mesosStatuses = map[string]string{
	"TASK_STAGING":          apiserver.TaskStatusPending,
	"TASK_STARTING":         apiserver.TaskStatusPending,
	"TASK_RUNNING":          apiserver.TaskStatusRunning,
	"TASK_KILLING":          apiserver.TaskStatusRunning,
	"TASK_STATUS_FINISHED":  apiserver.TaskStatusFinished,
	"TASK_STATUS_FAILED":    apiserver.TaskStatusFailed,
	"TASK_KILLED":           apiserver.TaskStatusFailed,
	"TASK_ERROR":            apiserver.TaskStatusFailed,
	"TASK_LOST":             apiserver.TaskStatusFailed,
	"TASK_DROPPED":          apiserver.TaskStatusFailed,
	"TASK_UNREACHABLE":      apiserver.TaskStatusFailed,
	"TASK_GONE":             apiserver.TaskStatusUnknown,
	"TASK_GONE_BY_OPERATOR": apiserver.TaskStatusUnknown,
	"TASK_STATUS_UNKNOWN":   apiserver.TaskStatusUnknown,
}

// Provider loads tasks from marathon
type Provider struct {
	url url.URL
	log *log.Entry
	cli *http.Client
}

// ProviderOptions keeps options for constructor
type ProviderOptions struct {
	Endpoint string
}

// NewProvider creates a new marathon driver instance using given options
func NewProvider(options ProviderOptions) (*Provider, error) {
	endpointURL, err := url.Parse(options.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("Could not parse endpoint url '%s' (%s)", options.Endpoint, err)
	}

	endpointURL.Path = strings.TrimSuffix(endpointURL.Path, "/")

	m := &Provider{
		url: *endpointURL,
		log: utils.NewLogEntry("api.marathon").WithField("url", endpointURL.String()),
		cli: &http.Client{
			Timeout: time.Duration(5 * time.Second),
		},
	}
	return m, nil
}

// LoadTasks from marathon
func (p *Provider) LoadTasks() ([]apiserver.Task, error) {
	endpoint := fmt.Sprintf("%s/apps?embed=apps.tasks", p.url.String())
	respApps := appsResponse{}

	response, err := p.cli.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("Could not load marathon apps (%s)", err)
	}

	if err := p.parseJSON(response, &respApps); err != nil {
		return nil, fmt.Errorf("Could not parse json response (%s)", err)
	}

	return p.buildTasks(respApps)
}

func (p *Provider) buildTasks(respApps appsResponse) ([]apiserver.Task, error) {
	result := make([]apiserver.Task, 0)

	for _, respApp := range respApps.Apps {
		instances := make([]apiserver.TaskInstance, 0)

		for _, respTask := range respApp.Tasks {
			instance := apiserver.TaskInstance{}
			instance.ID = respTask.ID
			instance.Addr = respTask.Host
			instance.Healthy = p.isTaskHealthy(respTask)
			instance.State = p.buildTaskStatus(respTask)
			instance.UpdatedAt = respTask.StartedAt
			instance.Payload = p.buildPayload(respTask)
			instances = append(instances, instance)
		}

		if len(instances) > 0 {
			task := apiserver.Task{}
			task.ID = respApp.ID
			task.Image = respApp.Container.Docker.Image
			task.CPU = respApp.CPU
			task.Mem = respApp.Mem
			task.Constraints = p.buildConstraints(respApp)
			task.UpdatedAt = respApp.VersionInfo.LastConfigChangeAt
			task.Instances = instances
			result = append(result, task)
		}
	}

	return result, nil
}
func (p *Provider) buildPayload(respTask taskResponse) payloads.Payload {
	return payloads.Payload{
		ContainerEnv: fmt.Sprintf("MESOS_TASK_ID=%s", respTask.ID),
	}
}

func (p *Provider) buildTaskStatus(respTask taskResponse) string {
	status := mesosStatuses[respTask.State]
	if status == "" {
		return apiserver.TaskStatusUnknown
	}
	return status
}

func (p *Provider) buildConstraints(respApp appResponse) map[string]string {
	constraints := map[string]string{}
	for _, constraint := range respApp.Constraints {
		if len(constraint) < 2 {
			continue
		}
		key := constraint[0]
		value := strings.Join(constraint[1:], ":")
		constraints[key] = value
	}
	return constraints
}

func (p *Provider) isTaskHealthy(respTask taskResponse) bool {
	if len(respTask.HealthCheckResults) == 0 {
		return false
	}

	for _, re := range respTask.HealthCheckResults {
		if !re.Alive {
			return false
		}
	}

	return true
}

func (p *Provider) parseJSON(response *http.Response, obj interface{}) error {
	if response.StatusCode != 200 {
		return fmt.Errorf("Unexpected response code, expected=200, actual=%d (%s)", response.StatusCode, response.Request.URL)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if err := json.Unmarshal(body, &obj); err != nil {
		return err
	}

	return nil
}
