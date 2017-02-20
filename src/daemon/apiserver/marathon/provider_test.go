package marathon

import (
	"daemon/apiserver"
	"daemon/payloads"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"runtime"
	"strconv"
	"testing"
)

func Test_Provider(t *testing.T) {

	t.Run("should load application from marathon", func(t *testing.T) {
		server := newTestMarathonServer(t, "apps.running.json")
		provider := newTestProvider(t, server.URL)

		providerTasks, err := provider.LoadTasks()
		require.NoError(t, err)
		require.Len(t, providerTasks, 1)

		task := providerTasks[0]
		require.Equal(t, "/app/demo", task.ID)
		require.Equal(t, "alpine", task.Image)
		require.Equal(t, float32(0.1), task.CPU)
		require.Equal(t, uint(128), task.Mem)
		require.Equal(t, map[string]string{"spot": "CLUSTER:0"}, task.Constraints)
		require.Equal(t, "2017-02-15 15:11:23.265 +0000 UTC", task.UpdatedAt.String())
		require.Len(t, task.Instances, 1)

		inst := task.Instances[0]
		require.Equal(t, "app_demo.27b10ccc-f395-11e6-9a83-424dbc3181a1", inst.ID)
		require.Equal(t, "10.1.1.244", inst.Addr.String())
		require.Equal(t, "2017-02-15 15:41:06.503 +0000 UTC", inst.UpdatedAt.String())
		require.Equal(t, true, inst.Healthy)
		require.Equal(t, apiserver.TaskStatusRunning, inst.State)
		require.Equal(t, payloads.Payload{ContainerEnv: "MESOS_TASK_ID=app_demo.27b10ccc-f395-11e6-9a83-424dbc3181a1"}, inst.Payload)
	})
}

func newTestProvider(t *testing.T, endpoint string) *Provider {
	provider, err := NewProvider(ProviderOptions{
		Endpoint: endpoint,
	})
	require.NoError(t, err)
	return provider
}

type testHTTPHandler struct {
	response io.Reader
}

func newTestMarathonServer(t *testing.T, fixtureName string) *httptest.Server {
	_, filename, _, ok := runtime.Caller(1)
	require.True(t, ok)

	fixturePath := path.Join(path.Base(filename), "../fixtures/", fixtureName)
	fixtureFile, err := os.Open(fixturePath)
	require.NoError(t, err)

	handler := &testHTTPHandler{
		response: fixtureFile,
	}

	mx := http.NewServeMux()
	mx.Handle("/v2/apps", handler)

	return httptest.NewServer(mx)
}

func (h *testHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bytes, err := ioutil.ReadAll(h.response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(bytes)))

	w.Write(bytes)
}
