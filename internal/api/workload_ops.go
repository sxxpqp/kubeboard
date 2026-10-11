package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func nowUnix() int64 { return time.Now().Unix() }

// workloadPods 按工作负载选择器列出其管理的 Pod
func (h *Handler) workloadPods(c *gin.Context) {
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
	for _, p := range pods.Items {
		containers := make([]string, 0, len(p.Spec.Containers))
		for _, ctr := range p.Spec.Containers {
			containers = append(containers, ctr.Name)
		}
		items = append(items, gin.H{
			"name": p.Name, "namespace": p.Namespace,
			"status": string(p.Status.Phase), "containers": containers,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

// scaleWorkload 调整 Deployment / StatefulSet 副本数
func (h *Handler) scaleWorkload(c *gin.Context) {
	var req struct {
		Kind      string `json:"kind" binding:"required"`
		Namespace string `json:"namespace" binding:"required"`
		Name      string `json:"name" binding:"required"`
		Replicas  int32  `json:"replicas"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	replicas := req.Replicas

	switch req.Kind {
	case "deployment":
		d, err := h.k8s.Clientset.AppsV1().Deployments(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		d.Spec.Replicas = &replicas
		if _, err := h.k8s.Clientset.AppsV1().Deployments(req.Namespace).Update(ctx, d, metav1.UpdateOptions{}); err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
	case "statefulset":
		s, err := h.k8s.Clientset.AppsV1().StatefulSets(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		s.Spec.Replicas = &replicas
		if _, err := h.k8s.Clientset.AppsV1().StatefulSets(req.Namespace).Update(ctx, s, metav1.UpdateOptions{}); err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "scale not supported for " + req.Kind})
		return
	}
	c.JSON(http.StatusOK, gin.H{"replicas": replicas})
}

// updateImage 更新工作负载中所有容器的镜像
func (h *Handler) updateImage(c *gin.Context) {
	var req struct {
		Kind      string `json:"kind" binding:"required"`
		Namespace string `json:"namespace" binding:"required"`
		Name      string `json:"name" binding:"required"`
		Image     string `json:"image" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()

	switch req.Kind {
	case "deployment":
		d, err := h.k8s.Clientset.AppsV1().Deployments(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		for i := range d.Spec.Template.Spec.Containers {
			d.Spec.Template.Spec.Containers[i].Image = req.Image
		}
		if _, err := h.k8s.Clientset.AppsV1().Deployments(req.Namespace).Update(ctx, d, metav1.UpdateOptions{}); err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
	case "statefulset":
		s, err := h.k8s.Clientset.AppsV1().StatefulSets(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		for i := range s.Spec.Template.Spec.Containers {
			s.Spec.Template.Spec.Containers[i].Image = req.Image
		}
		if _, err := h.k8s.Clientset.AppsV1().StatefulSets(req.Namespace).Update(ctx, s, metav1.UpdateOptions{}); err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "image update not supported for " + req.Kind})
		return
	}
	c.JSON(http.StatusOK, gin.H{"image": req.Image})
}

// rolloutRestart 通过重启注解触发滚动重启
func (h *Handler) rolloutRestart(c *gin.Context) {
	var req struct {
		Kind      string `json:"kind" binding:"required"`
		Namespace string `json:"namespace" binding:"required"`
		Name      string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	restartedAt := strconv.FormatInt(nowUnix(), 10)

	switch req.Kind {
	case "deployment":
		obj, err := h.k8s.Clientset.AppsV1().Deployments(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		if obj.Spec.Template.Annotations == nil {
			obj.Spec.Template.Annotations = map[string]string{}
		}
		obj.Spec.Template.Annotations["kubesphere.io/restartedAt"] = restartedAt
		if _, err := h.k8s.Clientset.AppsV1().Deployments(req.Namespace).Update(ctx, obj, metav1.UpdateOptions{}); err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
	case "statefulset":
		obj, err := h.k8s.Clientset.AppsV1().StatefulSets(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		if obj.Spec.Template.Annotations == nil {
			obj.Spec.Template.Annotations = map[string]string{}
		}
		obj.Spec.Template.Annotations["kubesphere.io/restartedAt"] = restartedAt
		if _, err := h.k8s.Clientset.AppsV1().StatefulSets(req.Namespace).Update(ctx, obj, metav1.UpdateOptions{}); err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
	case "daemonset":
		obj, err := h.k8s.Clientset.AppsV1().DaemonSets(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		if obj.Spec.Template.Annotations == nil {
			obj.Spec.Template.Annotations = map[string]string{}
		}
		obj.Spec.Template.Annotations["kubesphere.io/restartedAt"] = restartedAt
		if _, err := h.k8s.Clientset.AppsV1().DaemonSets(req.Namespace).Update(ctx, obj, metav1.UpdateOptions{}); err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "restart not supported for " + req.Kind})
		return
	}
	c.JSON(http.StatusOK, gin.H{"restarted": req.Name})
}

// cordonNode 封锁/解除封锁节点调度（对应 kubectl cordon/uncordon）
func (h *Handler) cordonNode(c *gin.Context) {
	var req struct {
		Name          string `json:"name" binding:"required"`
		Unschedulable bool   `json:"unschedulable"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()

	node, err := h.k8s.Clientset.CoreV1().Nodes().Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	node.Spec.Unschedulable = req.Unschedulable
	if _, err := h.k8s.Clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{}); err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"name": req.Name, "unschedulable": req.Unschedulable})
}

// pauseWorkload 暂停或恢复 Deployment 的滚动更新
func (h *Handler) pauseWorkload(c *gin.Context) {
	var req struct {
		Kind      string `json:"kind"`
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
		Paused    bool   `json:"paused"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	switch req.Kind {
	case "deployment":
		d, err := h.k8s.Clientset.AppsV1().Deployments(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		d.Spec.Paused = req.Paused
		if _, err := h.k8s.Clientset.AppsV1().Deployments(req.Namespace).Update(ctx, d, metav1.UpdateOptions{}); err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"paused": req.Paused})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "暂停/恢复仅支持 deployment"})
	}
}
