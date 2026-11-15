package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// summarizeContainerState 把容器状态收敛为 (state, reason, restarts, ready) 四元组。
// 映射规则：Running/Waiting/Terminated 三态 + 无状态时报 Unknown，供前端中文映射。
func summarizeContainerState(cs corev1.ContainerStatus) (state, reason string, restarts int32, ready bool) {
	ready = cs.Ready
	restarts = cs.RestartCount
	switch {
	case cs.State.Running != nil:
		return "Running", "", restarts, ready
	case cs.State.Waiting != nil:
		return "Waiting", cs.State.Waiting.Reason, restarts, ready
	case cs.State.Terminated != nil:
		return "Terminated", cs.State.Terminated.Reason, restarts, ready
	}
	return "Unknown", "", restarts, ready
}

// workloadContainers 列出工作负载下每个 Pod 的每个容器明细。
// 供「容器状态管理」区块使用：状态 / 重启次数 / 端口 / 探针 / 就绪。
// GET /api/workloads/containers?kind=&namespace=&name=
func (h *Handler) workloadContainers(c *gin.Context) {
	kind := c.Query("kind")
	ns := c.Query("namespace")
	name := c.Query("name")
	ctx := c.Request.Context()

	var selector *metav1.LabelSelector
	switch kind {
	case "deployment":
		d, err := h.k8s.Clientset.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		selector = d.Spec.Selector
	case "statefulset":
		s, err := h.k8s.Clientset.AppsV1().StatefulSets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		selector = s.Spec.Selector
	case "daemonset":
		ds, err := h.k8s.Clientset.AppsV1().DaemonSets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		selector = ds.Spec.Selector
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported kind: " + kind})
		return
	}

	sel, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	pods, err := h.k8s.Clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: sel.String()})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}

	items := make([]gin.H, 0, len(pods.Items))
	for i := range pods.Items {
		p := pods.Items[i]
		// 容器状态索引
		statusByName := map[string]corev1.ContainerStatus{}
		for _, cs := range p.Status.ContainerStatuses {
			statusByName[cs.Name] = cs
		}

		containers := make([]gin.H, 0, len(p.Spec.Containers))
		for _, ctr := range p.Spec.Containers {
			entry := gin.H{
				"name":  ctr.Name,
				"image": ctr.Image,
				"ready": false,
				"state": "Unknown",
			}
			// 端口
			ports := make([]gin.H, 0, len(ctr.Ports))
			for _, pt := range ctr.Ports {
				ports = append(ports, gin.H{
					"name": pt.Name, "containerPort": pt.ContainerPort, "protocol": string(pt.Protocol),
				})
			}
			entry["ports"] = ports
			// 探针（只报是否配置，悬停展示用）
			probes := make([]string, 0, 3)
			if ctr.LivenessProbe != nil {
				probes = append(probes, "liveness")
			}
			if ctr.ReadinessProbe != nil {
				probes = append(probes, "readiness")
			}
			if ctr.StartupProbe != nil {
				probes = append(probes, "startup")
			}
			entry["probes"] = probes

			if cs, ok := statusByName[ctr.Name]; ok {
				state, reason, restartCount, ready := summarizeContainerState(cs)
				entry["ready"] = ready
				entry["restartCount"] = restartCount
				entry["state"] = state
				switch state {
				case "Running":
					entry["startedAt"] = cs.State.Running.StartedAt.Unix()
				case "Waiting":
					entry["reason"] = reason
				case "Terminated":
					entry["reason"] = reason
					entry["exitCode"] = cs.State.Terminated.ExitCode
				}
			}
			containers = append(containers, entry)
		}

		ready := p.Status.Phase == corev1.PodRunning
		items = append(items, gin.H{
			"name":            p.Name,
			"namespace":       p.Namespace,
			"phase":           string(p.Status.Phase),
			"node":            p.Spec.NodeName,
			"podIP":           p.Status.PodIP,
			"restarts":        totalRestarts(p),
			"ready":           ready,
			"creationTime":    p.CreationTimestamp.Unix(),
			"containers":      containers,
		})
	}

	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}
