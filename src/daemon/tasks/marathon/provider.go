package marathon

import (
	"daemon/payloads"
	"daemon/tasks"
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
	"TASK_STAGING":          tasks.TaskStatusPending,
	"TASK_STARTING":         tasks.TaskStatusPending,
	"TASK_RUNNING":          tasks.TaskStatusRunning,
	"TASK_KILLING":          tasks.TaskStatusRunning,
	"TASK_STATUS_FINISHED":  tasks.TaskStatusFinished,
	"TASK_STATUS_FAILED":    tasks.TaskStatusFailed,
	"TASK_KILLED":           tasks.TaskStatusFailed,
	"TASK_ERROR":            tasks.TaskStatusFailed,
	"TASK_LOST":             tasks.TaskStatusFailed,
	"TASK_DROPPED":          tasks.TaskStatusFailed,
	"TASK_UNREACHABLE":      tasks.TaskStatusFailed,
	"TASK_GONE":             tasks.TaskStatusUnknown,
	"TASK_GONE_BY_OPERATOR": tasks.TaskStatusUnknown,
	"TASK_STATUS_UNKNOWN":   tasks.TaskStatusUnknown,
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

	m := &Provider{
		url: *endpointURL,
		log: utils.NewLogEntry("provider.marathon"),
		cli: &http.Client{
			Timeout: time.Duration(5 * time.Second),
		},
	}
	return m, nil
}

// LoadTasks from marathon
func (p *Provider) LoadTasks() ([]tasks.Task, error) {
	endpoint := fmt.Sprintf("%s/v2/apps?embed=apps.tasks", p.url.String())
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

func (p *Provider) buildTasks(respApps appsResponse) ([]tasks.Task, error) {
	result := make([]tasks.Task, 0)

	for _, respApp := range respApps.Apps {
		instances := make([]tasks.Instance, 0)

		for _, respTask := range respApp.Tasks {
			instance := tasks.Instance{}
			instance.ID = respTask.ID
			instance.Addr = respTask.Host
			instance.Healthy = p.isTaskHealthy(respTask)
			instance.State = p.buildTaskStatus(respTask)
			instance.UpdatedAt = respTask.StartedAt
			instance.Payload = p.buildPayload(respTask)
			instances = append(instances, instance)
		}

		if len(instances) > 0 {
			task := tasks.Task{}
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
		return tasks.TaskStatusUnknown
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
		return fmt.Errorf("Unexpected response code, expected=200, actual=%d", response.StatusCode)
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
