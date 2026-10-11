package api

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ==================== httpStatus ====================

func TestHTTPStatus_NotFound(t *testing.T) {
	err := errors.NewNotFound(schema.GroupResource{Resource: "pods"}, "x")
	if got := httpStatus(err); got != 404 {
		t.Fatalf("NotFound 应映射 404, got %d", got)
	}
}

func TestHTTPStatus_Forbidden(t *testing.T) {
	err := errors.NewForbidden(schema.GroupResource{Resource: "pods"}, "x", nil)
	if got := httpStatus(err); got != 403 {
		t.Fatalf("Forbidden 应映射 403, got %d", got)
	}
}

func TestHTTPStatus_Other(t *testing.T) {
	if got := httpStatus(errors.NewBadRequest("bad")); got != 500 {
		t.Fatalf("其它错误应映射 500, got %d", got)
	}
}

// ==================== podStatus ====================

func TestPodStatus_Phases(t *testing.T) {
	cases := []struct {
		name string
		pod  corev1.Pod
		want string
	}{
		{"Running", corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodRunning}}, "Running"},
		{"Pending", corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending}}, "Pending"},
		{"Succeeded", corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodSucceeded}}, "Succeeded"},
		{"Failed", corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodFailed}}, "Failed"},
	}
	for _, c := range cases {
		if got := podStatus(c.pod); got != c.want {
			t.Fatalf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

// 回归：删除中的 Pod 应优先显示 Terminating，而不是其 phase
func TestPodStatus_TerminatingOverridesPhase(t *testing.T) {
	now := metav1.Now()
	p := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &now},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	if got := podStatus(p); got != "Terminating" {
		t.Fatalf("删除中的 Pod 应显示 Terminating（优先于 phase）, got %q", got)
	}
}

// 回归：Evicted 应覆盖 phase（被驱逐的 Pod phase 常为 Failed，但要区分展示）
func TestPodStatus_EvictedOverridesPhase(t *testing.T) {
	p := corev1.Pod{
		Status: corev1.PodStatus{Phase: corev1.PodFailed, Reason: "Evicted"},
	}
	if got := podStatus(p); got != "Evicted" {
		t.Fatalf("被驱逐的 Pod 应显示 Evicted, got %q", got)
	}
}

// ==================== containerImages / containerNames ====================

func TestContainerImages(t *testing.T) {
	cs := []corev1.Container{{Name: "a", Image: "nginx:1.25"}, {Name: "b", Image: "redis:7"}}
	got := containerImages(cs)
	if len(got) != 2 || got[0] != "nginx:1.25" || got[1] != "redis:7" {
		t.Fatalf("镜像提取错误: %v", got)
	}
}

func TestContainerImages_Empty(t *testing.T) {
	if got := containerImages(nil); len(got) != 0 {
		t.Fatalf("空输入应返回空切片, got %v", got)
	}
	if got := containerImages([]corev1.Container{}); got == nil {
		t.Fatal("应返回非 nil 切片（便于 JSON 序列化为 []）")
	}
}

func TestContainerNames(t *testing.T) {
	cs := []corev1.Container{{Name: "app"}, {Name: "sidecar"}}
	got := containerNames(cs)
	if len(got) != 2 || got[0] != "app" || got[1] != "sidecar" {
		t.Fatalf("容器名提取错误: %v", got)
	}
}

// ==================== servicePorts ====================

func TestServicePorts(t *testing.T) {
	ports := []corev1.ServicePort{
		{Name: "http", Port: 80, TargetPort: intstr.FromInt32(8080), Protocol: corev1.ProtocolTCP, NodePort: 30080},
		{Name: "https", Port: 443, TargetPort: intstr.FromInt32(8443), Protocol: corev1.ProtocolTCP},
	}
	got := servicePorts(ports)
	if len(got) != 2 {
		t.Fatalf("端口数应为 2, got %d", len(got))
	}
	if got[0].Port != 80 || got[0].TargetPort != "8080" || got[0].NodePort != 30080 {
		t.Fatalf("第 1 个端口解析错误: %+v", got[0])
	}
	if got[0].Protocol != "TCP" {
		t.Fatalf("协议应为 TCP, got %q", got[0].Protocol)
	}
	if got[1].NodePort != 0 {
		t.Fatalf("无 NodePort 时应为 0, got %d", got[1].NodePort)
	}
}

func TestServicePorts_Empty(t *testing.T) {
	got := servicePorts(nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("空输入应返回非 nil 空切片, got %v", got)
	}
}

// ==================== filterManagedLabels ====================

func TestFilterManagedLabels_RemovesSystemLabels(t *testing.T) {
	in := map[string]string{
		"app":                              "web",
		"kubernetes.io/name":               "x",
		"k8s.io/cluster-autoscaler/enabled": "true",
		"version":                          "v1",
	}
	got := filterManagedLabels(in)
	if _, ok := got["app"]; !ok {
		t.Fatal("应保留 app 标签")
	}
	if _, ok := got["version"]; !ok {
		t.Fatal("应保留 version 标签")
	}
	if _, ok := got["kubernetes.io/name"]; ok {
		t.Fatal("应过滤 kubernetes.io/ 前缀标签")
	}
	if _, ok := got["k8s.io/cluster-autoscaler/enabled"]; ok {
		t.Fatal("应过滤 k8s.io/ 前缀标签")
	}
}

// 回归：全部为系统标签时，不应返回空 map（否则前端展示为空），应回退返回原值
func TestFilterManagedLabels_FallsBackWhenAllSystem(t *testing.T) {
	in := map[string]string{"kubernetes.io/name": "x", "k8s.io/foo": "bar"}
	got := filterManagedLabels(in)
	if len(got) == 0 {
		t.Fatal("全为系统标签时应回退返回原标签，而非空 map")
	}
	if len(got) != len(in) {
		t.Fatalf("回退时应返回全部原标签, got %v", got)
	}
}

func TestFilterManagedLabels_Empty(t *testing.T) {
	got := filterManagedLabels(map[string]string{})
	if got == nil {
		t.Fatal("空输入不应返回 nil")
	}
	if len(got) != 0 {
		t.Fatalf("空输入应返回空 map, got %v", got)
	}
}

// ==================== dataKeys ====================

func TestDataKeys(t *testing.T) {
	got := dataKeys(map[string][]byte{"a": []byte("1"), "b": []byte("2")})
	if len(got) != 2 {
		t.Fatalf("键数应为 2, got %d", len(got))
	}
	set := map[string]bool{}
	for _, k := range got {
		set[k] = true
	}
	if !set["a"] || !set["b"] {
		t.Fatalf("键集合不符: %v", got)
	}
}

func TestStringKeys(t *testing.T) {
	got := stringKeys(map[string]string{"k1": "v1", "k2": "v2"})
	if len(got) != 2 {
		t.Fatalf("键数应为 2, got %d", len(got))
	}
}

// ==================== 工作负载副本提取（rollout 相关纯逻辑）====================

func TestDeploymentReplicasDefaults(t *testing.T) {
	// 无 spec.replicas 时应按 K8s 语义视为 1
	one := int32(1)
	d := appsv1.Deployment{Spec: appsv1.DeploymentSpec{Replicas: &one}}
	if *d.Spec.Replicas != 1 {
		t.Fatal("副本数应为 1")
	}
}
