package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// metricsAvailable 报告集群是否真的装了 metrics-server。
func (h *Handler) metricsAvailable(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"available": h.k8s.Metrics != nil})
}

type nodeMetric struct {
	Name       string `json:"name"`
	CPUCores   string `json:"cpuCores"`
	CPUPercent int    `json:"cpuPercent"`
	MemBytes   string `json:"memBytes"`
	MemPercent int    `json:"memPercent"`
	Time       int64  `json:"time"`
}

// nodeMetrics 返回所有节点的即时用量与占容量百分比。
func (h *Handler) nodeMetrics(c *gin.Context) {
	if h.k8s.Metrics == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "metrics-server not installed"})
		return
	}
	ctx := c.Request.Context()

	// 容量用于计算百分比
	nodes, err := h.k8s.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	cpuCap := map[string]int64{}
	memCap := map[string]int64{}
	for _, n := range nodes.Items {
		if v, ok := n.Status.Capacity[corev1.ResourceCPU]; ok {
			cpuCap[n.Name] = v.MilliValue()
		}
		if v, ok := n.Status.Capacity[corev1.ResourceMemory]; ok {
			memCap[n.Name] = v.Value()
		}
	}

	list, err := h.k8s.Metrics.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	items := make([]nodeMetric, 0, len(list.Items))
	for _, m := range list.Items {
		cpu := m.Usage[corev1.ResourceCPU]
		mem := m.Usage[corev1.ResourceMemory]
		cpuMilli := cpu.MilliValue()
		memVal := mem.Value()
		nm := nodeMetric{
			Name:     m.Name,
			CPUCores: formatCores(cpuMilli),
			MemBytes: formatBytes(memVal),
			Time:     m.Timestamp.Unix(),
		}
		if cap := cpuCap[m.Name]; cap > 0 {
			nm.CPUPercent = int(cpuMilli * 100 / cap)
		}
		if cap := memCap[m.Name]; cap > 0 {
			nm.MemPercent = int(memVal * 100 / cap)
		}
		items = append(items, nm)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

// podMetrics 返回指定命名空间内 Pod 的用量（namespace 为空则全集群）。
func (h *Handler) podMetrics(c *gin.Context) {
	if h.k8s.Metrics == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "metrics-server not installed"})
		return
	}
	ns := c.Query("namespace")
	ctx := c.Request.Context()

	list, err := h.k8s.Metrics.MetricsV1beta1().PodMetricses(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(list.Items))
	for _, m := range list.Items {
		var cpuMilli, memVal int64
		containers := make([]gin.H, 0, len(m.Containers))
		for _, ctr := range m.Containers {
			c := ctr.Usage[corev1.ResourceCPU]
			mem := ctr.Usage[corev1.ResourceMemory]
			cpuMilli += c.MilliValue()
			memVal += mem.Value()
			containers = append(containers, gin.H{
				"name":     ctr.Name,
				"cpuCores": formatCores(c.MilliValue()),
				"memBytes": formatBytes(mem.Value()),
			})
		}
		items = append(items, gin.H{
			"namespace":     m.Namespace,
			"name":          m.Name,
			"cpuCores":      formatCores(cpuMilli),
			"memBytes":      formatBytes(memVal),
			"cpuMilli":      cpuMilli,
			"memBytesRaw":   memVal,
			"containers":    containers,
			"time":          m.Timestamp.Unix(),
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func formatCores(milli int64) string {
	return resource.NewMilliQuantity(milli, resource.DecimalSI).String()
}

func formatBytes(b int64) string {
	return resource.NewQuantity(b, resource.BinarySI).String()
}
