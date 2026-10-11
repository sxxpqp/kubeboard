package api

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func nodeRoles(labels map[string]string) []string {
	roles := []string{}
	for k := range labels {
		if strings.HasPrefix(k, "node-role.kubernetes.io/") {
			roles = append(roles, strings.TrimPrefix(k, "node-role.kubernetes.io/"))
		}
	}
	if len(roles) == 0 {
		roles = append(roles, "worker")
	}
	sort.Strings(roles)
	return roles
}

func nodeInternalIP(addrs []metav1AddressAlias) string {
	for _, a := range addrs {
		if a.Type == "InternalIP" {
			return a.Address
		}
	}
	return ""
}

func (h *Handler) listNamespaces(c *gin.Context) {
	nsList, err := h.k8s.Clientset.CoreV1().Namespaces().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(nsList.Items))
	for _, ns := range nsList.Items {
		items = append(items, gin.H{
			"name":         ns.Name,
			"status":       string(ns.Status.Phase),
			"labels":       filterManagedLabels(ns.Labels),
			"creationTime": ns.CreationTimestamp.Unix(),
		})
	}
	c.JSON(200, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) listWorkloads(c *gin.Context) {
	ctx := c.Request.Context()
	kind := c.DefaultQuery("kind", "deployment")
	ns := c.Query("namespace")

	items := []gin.H{}
	var total int
	var err error

	switch kind {
	case "deployment":
		var l *appsv1DeployList
		l, err = h.k8s.Clientset.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
		if err == nil {
			total = len(l.Items)
			for _, d := range l.Items {
				items = append(items, gin.H{
					"namespace": d.Namespace, "name": d.Name,
					"ready":   fmt.Sprintf("%d/%d", d.Status.ReadyReplicas, *d.Spec.Replicas),
					"replicas": d.Status.Replicas, "available": d.Status.AvailableReplicas,
					"images":    containerImages(d.Spec.Template.Spec.Containers),
					"creationTime": d.CreationTimestamp.Unix(),
				})
			}
		}
	case "statefulset":
		var l *appsv1StatefulSetList
		l, err = h.k8s.Clientset.AppsV1().StatefulSets(ns).List(ctx, metav1.ListOptions{})
		if err == nil {
			total = len(l.Items)
			for _, d := range l.Items {
				items = append(items, gin.H{
					"namespace": d.Namespace, "name": d.Name,
					"ready":   fmt.Sprintf("%d/%d", d.Status.ReadyReplicas, *d.Spec.Replicas),
					"replicas": d.Status.Replicas,
					"images":    containerImages(d.Spec.Template.Spec.Containers),
					"creationTime": d.CreationTimestamp.Unix(),
				})
			}
		}
	case "daemonset":
		var l *appsv1DaemonSetList
		l, err = h.k8s.Clientset.AppsV1().DaemonSets(ns).List(ctx, metav1.ListOptions{})
		if err == nil {
			total = len(l.Items)
			for _, d := range l.Items {
				items = append(items, gin.H{
					"namespace": d.Namespace, "name": d.Name,
					"ready":   fmt.Sprintf("%d/%d", d.Status.NumberReady, d.Status.DesiredNumberScheduled),
					"desired": d.Status.DesiredNumberScheduled,
					"images":    containerImages(d.Spec.Template.Spec.Containers),
					"creationTime": d.CreationTimestamp.Unix(),
				})
			}
		}
	case "job":
		var l *batchv1.JobList
		l, err = h.k8s.Clientset.BatchV1().Jobs(ns).List(ctx, metav1.ListOptions{})
		if err == nil {
			total = len(l.Items)
			for _, j := range l.Items {
				items = append(items, gin.H{
					"namespace": j.Namespace, "name": j.Name,
					"ready":   fmt.Sprintf("%d/%d", j.Status.Succeeded, derefInt32(j.Spec.Completions)),
					"completions": derefInt32(j.Spec.Completions),
					"images":    containerImages(j.Spec.Template.Spec.Containers),
					"creationTime": j.CreationTimestamp.Unix(),
				})
			}
		}
	case "cronjob":
		var l *batchv1.CronJobList
		l, err = h.k8s.Clientset.BatchV1().CronJobs(ns).List(ctx, metav1.ListOptions{})
		if err == nil {
			total = len(l.Items)
			for _, cj := range l.Items {
				items = append(items, gin.H{
					"namespace": cj.Namespace, "name": cj.Name,
					"schedule": cj.Spec.Schedule,
					"suspend":  cj.Spec.Suspend != nil && *cj.Spec.Suspend,
					"active":   len(cj.Status.Active),
					"images":    containerImages(cj.Spec.JobTemplate.Spec.Template.Spec.Containers),
					"creationTime": cj.CreationTimestamp.Unix(),
				})
			}
		}
	default:
		c.JSON(400, gin.H{"error": "unsupported workload kind: " + kind})
		return
	}
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total})
}

func (h *Handler) listPods(c *gin.Context) {
	ns := c.Query("namespace")
	nodeFilter := c.Query("node")
	pods, err := h.k8s.Clientset.CoreV1().Pods(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(pods.Items))
	for _, p := range pods.Items {
		if nodeFilter != "" && p.Spec.NodeName != nodeFilter {
			continue
		}
		restarts := int32(0)
		for _, cs := range p.Status.ContainerStatuses {
			restarts += cs.RestartCount
		}
		items = append(items, gin.H{
			"namespace": p.Namespace, "name": p.Name,
			"status":   podStatus(p),
			"restarts": restarts,
			"node":     p.Spec.NodeName,
			"ip":       p.Status.PodIP,
			"images":   containerImages(p.Spec.Containers),
			"containerNames": containerNames(p.Spec.Containers),
			"creationTime": p.CreationTimestamp.Unix(),
		})
	}
	c.JSON(200, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) listServices(c *gin.Context) {
	ns := c.Query("namespace")
	svcs, err := h.k8s.Clientset.CoreV1().Services(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(svcs.Items))
	for _, s := range svcs.Items {
		items = append(items, gin.H{
			"namespace": s.Namespace, "name": s.Name,
			"type":         string(s.Spec.Type),
			"clusterIP":    s.Spec.ClusterIP,
			"externalIP":   externalIP(s.Spec.ExternalIPs),
			"ports":        servicePorts(s.Spec.Ports),
			"creationTime": s.CreationTimestamp.Unix(),
		})
	}
	c.JSON(200, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) listConfigMaps(c *gin.Context) {
	ns := c.Query("namespace")
	list, err := h.k8s.Clientset.CoreV1().ConfigMaps(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(list.Items))
	for _, cm := range list.Items {
		items = append(items, gin.H{
			"namespace": cm.Namespace, "name": cm.Name,
			"keys":         stringKeys(cm.Data),
			"creationTime": cm.CreationTimestamp.Unix(),
		})
	}
	c.JSON(200, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) listSecrets(c *gin.Context) {
	ns := c.Query("namespace")
	list, err := h.k8s.Clientset.CoreV1().Secrets(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(list.Items))
	for _, s := range list.Items {
		items = append(items, gin.H{
			"namespace": s.Namespace, "name": s.Name,
			"type":         string(s.Type),
			"keys":         dataKeys(s.Data),
			"creationTime": s.CreationTimestamp.Unix(),
		})
	}
	c.JSON(200, gin.H{"items": items, "total": len(items)})
}

func (h *Handler) listEvents(c *gin.Context) {
	ns := c.Query("namespace")
	list, err := h.k8s.Clientset.CoreV1().Events(ns).List(c.Request.Context(), metav1.ListOptions{Limit: 200})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	sort.Slice(list.Items, func(i, j int) bool {
		return list.Items[i].LastTimestamp.Time.After(list.Items[j].LastTimestamp.Time)
	})
	items := make([]gin.H, 0, len(list.Items))
	for _, e := range list.Items {
		items = append(items, gin.H{
			"namespace":  e.Namespace,
			"type":       e.Type,
			"reason":     e.Reason,
			"message":    e.Message,
			"object":     fmt.Sprintf("%s/%s", e.InvolvedObject.Kind, e.InvolvedObject.Name),
			"kind":       e.InvolvedObject.Kind,
			"name":       e.InvolvedObject.Name,
			"uid":        string(e.InvolvedObject.UID),
			"apiVersion": e.InvolvedObject.APIVersion,
			"count":      e.Count,
			"lastSeen":   e.LastTimestamp.Unix(),
		})
	}
	c.JSON(200, gin.H{"items": items, "total": len(items)})
}
