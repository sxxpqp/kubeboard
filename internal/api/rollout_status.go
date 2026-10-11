package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// workloadRolloutStatus 返回 Deployment/StatefulSet 的滚动更新状态
func (h *Handler) workloadRolloutStatus(c *gin.Context) {
	kind := c.Query("kind")
	ns := c.Query("namespace")
	name := c.Query("name")
	ctx := c.Request.Context()

	switch kind {
	case "deployment":
		d, err := h.k8s.Clientset.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		// 是否处于滚动更新中
		updated := d.Status.UpdatedReplicas
		replicas := d.Status.Replicas
		ready := d.Status.ReadyReplicas
		available := d.Status.AvailableReplicas
		inProgress := updated < replicas || ready < replicas ||
			d.Status.ObservedGeneration < d.Generation

		// 策略
		strategyType := string(d.Spec.Strategy.Type)
		var maxSurge, maxUnavailable string
		if d.Spec.Strategy.RollingUpdate != nil {
			if d.Spec.Strategy.RollingUpdate.MaxSurge != nil {
				maxSurge = d.Spec.Strategy.RollingUpdate.MaxSurge.String()
			}
			if d.Spec.Strategy.RollingUpdate.MaxUnavailable != nil {
				maxUnavailable = d.Spec.Strategy.RollingUpdate.MaxUnavailable.String()
			}
		}

		// 进度百分比
		progress := 0
		desired := int32(0)
		if d.Spec.Replicas != nil {
			desired = *d.Spec.Replicas
		}
		if desired > 0 {
			progress = int(float64(updated) / float64(desired) * 100)
			if progress > 100 {
				progress = 100
			}
		} else {
			progress = 100
		}

		// 条件
		conds := make([]gin.H, 0, len(d.Status.Conditions))
		for _, c := range d.Status.Conditions {
			conds = append(conds, gin.H{
				"type":    string(c.Type),
				"status":  string(c.Status),
				"reason":  c.Reason,
				"message": c.Message,
				"lastUpdateTime": c.LastUpdateTime.Unix(),
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"kind": kind, "namespace": ns, "name": name,
			"strategy":         strategyType,
			"maxSurge":         maxSurge,
			"maxUnavailable":   maxUnavailable,
			"desired":          desired,
			"replicas":         replicas,
			"updatedReplicas":  updated,
			"readyReplicas":    ready,
			"availableReplicas": available,
			"unavailableReplicas": d.Status.UnavailableReplicas,
			"observedGeneration":  d.Status.ObservedGeneration,
			"generation":          d.Generation,
			"progress":            progress,
			"inProgress":          inProgress,
			"paused":              d.Spec.Paused,
			"conditions":          conds,
			"revision":            d.Annotations[revisionAnnotation],
			"message":             rolloutMessage(d, inProgress),
		})

	case "statefulset":
		s, err := h.k8s.Clientset.AppsV1().StatefulSets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		desired := int32(0)
		if s.Spec.Replicas != nil {
			desired = *s.Spec.Replicas
		}
		updated := s.Status.UpdatedReplicas
		ready := s.Status.ReadyReplicas
		progress := 100
		if desired > 0 {
			progress = int(float64(updated) / float64(desired) * 100)
			if progress > 100 {
				progress = 100
			}
		}
		inProgress := updated < desired || ready < desired ||
			s.Status.CurrentRevision != s.Status.UpdateRevision

		strategyType := string(s.Spec.UpdateStrategy.Type)
		var partition string
		if s.Spec.UpdateStrategy.RollingUpdate != nil && s.Spec.UpdateStrategy.RollingUpdate.Partition != nil {
			partition = itoa(int64(*s.Spec.UpdateStrategy.RollingUpdate.Partition))
		}

		conds := make([]gin.H, 0, len(s.Status.Conditions))
		for _, c := range s.Status.Conditions {
			conds = append(conds, gin.H{
				"type": string(c.Type), "status": string(c.Status),
				"reason": c.Reason, "message": c.Message,
				"lastUpdateTime": c.LastTransitionTime.Unix(),
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"kind": kind, "namespace": ns, "name": name,
			"strategy":       strategyType,
			"partition":      partition,
			"desired":        desired,
			"replicas":       s.Status.Replicas,
			"updatedReplicas": updated,
			"readyReplicas":  ready,
			"availableReplicas": s.Status.AvailableReplicas,
			"currentReplicas": s.Status.CurrentReplicas,
			"progress":        progress,
			"inProgress":      inProgress,
			"paused":          false,
			"currentRevision": s.Status.CurrentRevision,
			"updateRevision":  s.Status.UpdateRevision,
			"conditions":      conds,
			"message":         stsRolloutMessage(s, inProgress),
		})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 deployment / statefulset"})
	}
}

func rolloutMessage(d *appsv1.Deployment, inProgress bool) string {
	if d.Spec.Paused {
		return "滚动更新已暂停"
	}
	if inProgress {
		return "正在滚动更新"
	}
	return "滚动更新完成"
}

func stsRolloutMessage(s *appsv1.StatefulSet, inProgress bool) string {
	if inProgress {
		return "正在滚动更新"
	}
	return "滚动更新完成"
}
