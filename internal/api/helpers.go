package api

import (
	"fmt"
	"net/http"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

// 以下别名把 handler 中的泛化写法映射到真实 K8s 类型
type metav1AddressAlias = corev1.NodeAddress
type appsv1DeployList = appsv1.DeploymentList
type appsv1StatefulSetList = appsv1.StatefulSetList
type appsv1DaemonSetList = appsv1.DaemonSetList

func httpStatus(err error) int {
	if errors.IsNotFound(err) {
		return http.StatusNotFound
	}
	if errors.IsForbidden(err) {
		return http.StatusForbidden
	}
	return http.StatusInternalServerError
}

func podStatus(p corev1.Pod) string {
	if p.DeletionTimestamp != nil {
		return "Terminating"
	}
	if p.Status.Reason == "Evicted" {
		return "Evicted"
	}
	return string(p.Status.Phase)
}

func containerImages(cs []corev1.Container) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.Image)
	}
	return out
}

func containerNames(cs []corev1.Container) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.Name)
	}
	return out
}

func externalIPs(ips []string) string {
	return fmt.Sprintf("%v", ips)
}

func servicePorts(ports []corev1.ServicePort) []ginPort {
	out := make([]ginPort, 0, len(ports))
	for _, p := range ports {
		out = append(out, ginPort{
			Port: int(p.Port), Protocol: string(p.Protocol),
			NodePort: int(p.NodePort), TargetPort: p.TargetPort.String(),
		})
	}
	return out
}

type ginPort struct {
	Port       int    `json:"port"`
	Protocol   string `json:"protocol"`
	NodePort   int    `json:"nodePort,omitempty"`
	TargetPort string `json:"targetPort,omitempty"`
}

func dataKeys(data map[string][]byte) []string {
	out := make([]string, 0, len(data))
	for k := range data {
		out = append(out, k)
	}
	return out
}

func stringKeys(data map[string]string) []string {
	out := make([]string, 0, len(data))
	for k := range data {
		out = append(out, k)
	}
	return out
}

func filterManagedLabels(labels map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range labels {
		if !strings.HasPrefix(k, "kubernetes.io/") && !strings.HasPrefix(k, "k8s.io/") {
			out[k] = v
		}
	}
	if len(out) == 0 {
		out = labels
	}
	return out
}

func externalIP(ips []string) string { return externalIPs(ips) }
