package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sxxpqp/kubeboard/internal/k8s"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type revisionUITransport func(*http.Request) (*http.Response, error)

func (transport revisionUITransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestRevisionUIIncludesTemplateAndExcludesUnownedReplicaSets(t *testing.T) {
	desired := int32(1)
	controller := true
	deployment := &appsv1.Deployment{TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"}, ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default", UID: types.UID("app-uid"), Annotations: map[string]string{revisionAnnotation: "2"}}, Spec: appsv1.DeploymentSpec{Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "app"}}}}
	replica := appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "app-rs", Namespace: "default", Annotations: map[string]string{revisionAnnotation: "2"}, OwnerReferences: []metav1.OwnerReference{{Kind: "Deployment", Name: "app", UID: deployment.UID, Controller: &controller}}}, Spec: appsv1.ReplicaSetSpec{Replicas: &desired, Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "nginx:1.27.5", Command: []string{"/bin/sh", "-c"}, Args: []string{"echo hello world"}, Env: []corev1.EnvVar{{Name: "TOKEN", ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "token"}, Key: "value"}}}}}}}}}}
	unowned := *replica.DeepCopy()
	unowned.Name = "other-rs"
	unowned.OwnerReferences[0].UID = "different-uid"
	unowned.Annotations[revisionAnnotation] = "99"
	writes := 0
	client, err := kubernetes.NewForConfig(&rest.Config{Host: "https://revision-ui.test", Transport: revisionUITransport(func(request *http.Request) (*http.Response, error) {
		if request.Method != "GET" {
			writes++
			t.Errorf("unexpected mutation %s", request.Method)
		}
		var body interface{}
		if strings.HasSuffix(request.URL.Path, "/deployments/app") {
			body = deployment
		} else {
			body = &appsv1.ReplicaSetList{TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "ReplicaSetList"}, Items: []appsv1.ReplicaSet{replica, unowned}}
		}
		data, _ := json.Marshal(body)
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(data))}, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	handler := &Handler{k8s: &k8s.Client{Clientset: client}}
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Request = httptest.NewRequest("GET", "/workloads/revisions?kind=deployment&namespace=default&name=app", nil)
	handler.workloadRevisions(ctx)
	if response.Code != 200 {
		t.Fatalf("history failed: %s", response.Body.String())
	}
	var history struct {
		Items []revisionInfo `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &history); err != nil {
		t.Fatal(err)
	}
	if len(history.Items) != 1 || history.Items[0].Name != "app-rs" {
		t.Fatalf("wrong ownership selection: %+v", history.Items)
	}
	template := history.Items[0].Template
	if template == nil || template.Spec.Containers[0].Args[0] != "echo hello world" || template.Spec.Containers[0].Env[0].ValueFrom.SecretKeyRef.Name != "token" {
		t.Fatalf("incomplete revision template: %+v", template)
	}
	response = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(response)
	ctx.Request = httptest.NewRequest("POST", "/workloads/rollback", strings.NewReader(`{"kind":"deployment","namespace":"default","name":"app","revision":99}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	handler.rolloutUndo(ctx)
	if response.Code != 404 || writes != 0 {
		t.Fatalf("must not roll back to an unowned replica set: status=%d writes=%d", response.Code, writes)
	}
}
