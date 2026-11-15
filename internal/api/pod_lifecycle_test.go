package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sxxpqp/kubeboard/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestPodDetailIncludesInitContainerFailures(t *testing.T) {
	pod := corev1.Pod{Spec: corev1.PodSpec{
		InitContainers: []corev1.Container{{Name: "setup", Image: "setup:v1"}},
		Containers:     []corev1.Container{{Name: "app", Image: "app:v1"}},
	}, Status: corev1.PodStatus{
		InitContainerStatuses: []corev1.ContainerStatus{{Name: "setup", RestartCount: 3,
			State:                corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Error", ExitCode: 1, FinishedAt: metav1.Unix(30, 0)}},
			LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled", ExitCode: 137, FinishedAt: metav1.Unix(20, 0)}},
		}},
		ContainerStatuses: []corev1.ContainerStatus{{Name: "app", RestartCount: 2}},
	}}
	client, err := kubernetes.NewForConfig(&rest.Config{Host: "https://lifecycle.test", Transport: revisionUITransport(func(r *http.Request) (*http.Response, error) {
		if r.Method != "GET" {
			t.Fatalf("unexpected mutation: %s", r.Method)
		}
		data, _ := json.Marshal(pod)
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(data))}, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Request = httptest.NewRequest("GET", "/pods/detail?namespace=default&name=app", nil)
	(&Handler{k8s: &k8s.Client{Clientset: client}}).podDetail(ctx)
	if response.Code != 200 {
		t.Fatalf("%d: %s", response.Code, response.Body.String())
	}
	var result struct {
		Restarts   int32 `json:"restarts"`
		Containers []struct {
			Name           string `json:"name"`
			State          string `json:"state"`
			ExitCode       int    `json:"exitCode"`
			FinishedAt     int64  `json:"finishedAt"`
			LastTerminated struct {
				Reason string `json:"reason"`
			} `json:"lastTerminated"`
		} `json:"containers"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Restarts != 5 || len(result.Containers) != 2 {
		t.Fatalf("unexpected detail: %s", response.Body.String())
	}
	setup := result.Containers[0]
	if setup.Name != "setup" || setup.State != "Terminated" || setup.ExitCode != 1 || setup.FinishedAt != 30 || setup.LastTerminated.Reason != "OOMKilled" {
		t.Fatalf("missing init failure: %s", response.Body.String())
	}
	if totalRestarts(corev1.Pod{}) != 0 {
		t.Fatal("empty pod restart count must be zero")
	}
}
