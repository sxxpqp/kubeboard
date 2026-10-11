package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// podDetail 返回 Pod 的完整生命周期信息（等价 kubectl describe pod 的关键部分）
func (h *Handler) podDetail(c *gin.Context) {
	ns := c.Query("namespace")
	name := c.Query("name")
	if ns == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "namespace and name required"})
		return
	}
	p, err := h.k8s.Clientset.CoreV1().Pods(ns).Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}

	// 条件链
	conditions := make([]gin.H, 0, len(p.Status.Conditions))
	for _, c := range p.Status.Conditions {
		conditions = append(conditions, gin.H{
			"type":   string(c.Type),
			"status": string(c.Status),
			"reason": c.Reason,
			"message": c.Message,
			"lastTransitionTime": c.LastTransitionTime.Unix(),
		})
	}

	// 每容器状态与启动时间
	containers := make([]gin.H, 0, len(p.Spec.Containers))
	for _, ctr := range p.Spec.Containers {
		entry := gin.H{
			"name":  ctr.Name,
			"image": ctr.Image,
			"ready": false,
			"state": "Unknown",
		}
		for _, cs := range p.Status.ContainerStatuses {
			if cs.Name != ctr.Name {
				continue
			}
			entry["ready"] = cs.Ready
			entry["restartCount"] = cs.RestartCount
			entry["imageID"] = cs.ImageID
			switch {
			case cs.State.Running != nil:
				entry["state"] = "Running"
				entry["startedAt"] = cs.State.Running.StartedAt.Unix()
			case cs.State.Waiting != nil:
				entry["state"] = "Waiting"
				entry["reason"] = cs.State.Waiting.Reason
				entry["message"] = cs.State.Waiting.Message
			case cs.State.Terminated != nil:
				entry["state"] = "Terminated"
				entry["reason"] = cs.State.Terminated.Reason
				entry["message"] = cs.State.Terminated.Message
				entry["startedAt"] = cs.State.Terminated.StartedAt.Unix()
				entry["finishedAt"] = cs.State.Terminated.FinishedAt.Unix()
				entry["exitCode"] = cs.State.Terminated.ExitCode
			}
			// 最后终止状态（重启前的）
			if cs.LastTerminationState.Terminated != nil {
				entry["lastTerminated"] = gin.H{
					"reason":     cs.LastTerminationState.Terminated.Reason,
					"exitCode":   cs.LastTerminationState.Terminated.ExitCode,
					"finishedAt": cs.LastTerminationState.Terminated.FinishedAt.Unix(),
				}
			}
			break
		}
		containers = append(containers, entry)
	}

	// 关键时间点
	startTime := int64(0)
	if p.Status.StartTime != nil {
		startTime = p.Status.StartTime.Unix()
	}

	// QoS 与调度信息
	c.JSON(http.StatusOK, gin.H{
		"namespace":    p.Namespace,
		"name":         p.Name,
		"phase":        string(p.Status.Phase),
		"status":       podStatus(*p),
		"startTime":    startTime,
		"nodeName":     p.Spec.NodeName,
		"podIP":        p.Status.PodIP,
		"hostIP":       p.Status.HostIP,
		"qosClass":     string(p.Status.QOSClass),
		"restartPolicy": string(p.Spec.RestartPolicy),
		"schedulerName": p.Spec.SchedulerName,
		"serviceAccount": p.Spec.ServiceAccountName,
		"conditions":   conditions,
		"containers":   containers,
		"labels":       p.Labels,
		"annotations":  p.Annotations,
		"creationTime": p.CreationTimestamp.Unix(),
		"deletionTime": deletionTime(p),
		"restarts":     totalRestarts(*p),
	})
}

func deletionTime(p *corev1.Pod) int64 {
	if p.DeletionTimestamp == nil {
		return 0
	}
	return p.DeletionTimestamp.Unix()
}

func totalRestarts(p corev1.Pod) int32 {
	var n int32
	for _, cs := range p.Status.ContainerStatuses {
		n += cs.RestartCount
	}
	return n
}
