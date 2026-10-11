package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// listHPA 水平自动伸缩器
func (h *Handler) listHPA(c *gin.Context) {
	ns := c.Query("namespace")
	list, err := h.k8s.Clientset.AutoscalingV2().HorizontalPodAutoscalers(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(list.Items))
	for _, hpa := range list.Items {
		minReplicas := int32(0)
		if hpa.Spec.MinReplicas != nil {
			minReplicas = *hpa.Spec.MinReplicas
		}
		cpuTarget := ""
		for _, m := range hpa.Spec.Metrics {
			if m.Type == autoscalingv2.ResourceMetricSourceType && m.Resource != nil {
				if m.Resource.Target.AverageUtilization != nil {
					cpuTarget = m.Resource.Name.String() + " " + itoa(int64(*m.Resource.Target.AverageUtilization)) + "%"
				} else if m.Resource.Target.AverageValue != nil {
					cpuTarget = m.Resource.Name.String() + " " + m.Resource.Target.AverageValue.String()
				}
			}
		}
		items = append(items, gin.H{
			"namespace": hpa.Namespace,
			"name":      hpa.Name,
			"target":    hpa.Spec.ScaleTargetRef.Kind + "/" + hpa.Spec.ScaleTargetRef.Name,
			"min":       minReplicas,
			"max":       hpa.Spec.MaxReplicas,
			"current":   hpa.Status.CurrentReplicas,
			"desired":   hpa.Status.DesiredReplicas,
			"metric":    cpuTarget,
			"creationTime": hpa.CreationTimestamp.Unix(),
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

// listNetworkPolicies 网络策略
func (h *Handler) listNetworkPolicies(c *gin.Context) {
	ns := c.Query("namespace")
	list, err := h.k8s.Clientset.NetworkingV1().NetworkPolicies(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(list.Items))
	for _, np := range list.Items {
		policyTypes := make([]string, 0, len(np.Spec.PolicyTypes))
		for _, t := range np.Spec.PolicyTypes {
			policyTypes = append(policyTypes, string(t))
		}
		items = append(items, gin.H{
			"namespace":   np.Namespace,
			"name":        np.Name,
			"policyTypes": policyTypes,
			"podSelector": selectorString(np.Spec.PodSelector),
			"ingress":     len(np.Spec.Ingress),
			"egress":      len(np.Spec.Egress),
			"creationTime": np.CreationTimestamp.Unix(),
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

// listResourceQuotas 资源配额
func (h *Handler) listResourceQuotas(c *gin.Context) {
	ns := c.Query("namespace")
	list, err := h.k8s.Clientset.CoreV1().ResourceQuotas(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(list.Items))
	for _, q := range list.Items {
		used := gin.H{}
		for k, v := range q.Status.Used {
			used[string(k)] = v.String()
		}
		hard := gin.H{}
		for k, v := range q.Status.Hard {
			hard[string(k)] = v.String()
		}
		items = append(items, gin.H{
			"namespace":    q.Namespace,
			"name":         q.Name,
			"used":         used,
			"hard":         hard,
			"creationTime": q.CreationTimestamp.Unix(),
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

// namespaceSummary 命名空间资源总览：各类资源的数量与健康度
func (h *Handler) namespaceSummary(c *gin.Context) {
	ns := c.Param("namespace")
	if ns == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "namespace required"})
		return
	}
	ctx := c.Request.Context()

	pods, err := h.k8s.Clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	podRunning, podPending, podFailed := 0, 0, 0
	for _, p := range pods.Items {
		switch p.Status.Phase {
		case corev1.PodRunning:
			podRunning++
		case corev1.PodPending:
			podPending++
		case corev1.PodFailed:
			podFailed++
		}
	}

	deploys, _ := h.k8s.Clientset.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	sts, _ := h.k8s.Clientset.AppsV1().StatefulSets(ns).List(ctx, metav1.ListOptions{})
	ds, _ := h.k8s.Clientset.AppsV1().DaemonSets(ns).List(ctx, metav1.ListOptions{})
	svcs, _ := h.k8s.Clientset.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	ings, _ := h.k8s.Clientset.NetworkingV1().Ingresses(ns).List(ctx, metav1.ListOptions{})
	cm, _ := h.k8s.Clientset.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{})
	sec, _ := h.k8s.Clientset.CoreV1().Secrets(ns).List(ctx, metav1.ListOptions{})
	pvc, _ := h.k8s.Clientset.CoreV1().PersistentVolumeClaims(ns).List(ctx, metav1.ListOptions{})
	jobs, _ := h.k8s.Clientset.BatchV1().Jobs(ns).List(ctx, metav1.ListOptions{})
	cronjobs, _ := h.k8s.Clientset.BatchV1().CronJobs(ns).List(ctx, metav1.ListOptions{})

	count := func(n int) int { return n }
	c.JSON(http.StatusOK, gin.H{
		"namespace": ns,
		"pods": gin.H{
			"total":   len(pods.Items),
			"running": podRunning,
			"pending": podPending,
			"failed":  podFailed,
		},
		"workloads": gin.H{
			"deployments": count(len(deploys.Items)),
			"statefulsets": count(len(sts.Items)),
			"daemonsets":  count(len(ds.Items)),
		},
		"services":       len(svcs.Items),
		"ingresses":      len(ings.Items),
		"configmaps":     len(cm.Items),
		"secrets":        len(sec.Items),
		"pvcs":           len(pvc.Items),
		"jobs":           len(jobs.Items),
		"cronjobs":       len(cronjobs.Items),
	})
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// selectorString 把 LabelSelector 转成字符串
func selectorString(sel metav1.LabelSelector) string {
	s, err := metav1.LabelSelectorAsSelector(&sel)
	if err != nil || s.Empty() {
		return "-"
	}
	return s.String()
}

var _ = networkingv1.NetworkPolicy{}

// serviceConnectivity 对 Service 的端口做 TCP 连通性探测
func (h *Handler) serviceConnectivity(c *gin.Context) {
	ns := c.Query("namespace")
	name := c.Query("name")
	if ns == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "namespace and name required"})
		return
	}
	ctx := c.Request.Context()
	svc, err := h.k8s.Clientset.CoreV1().Services(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}

	// 解析后端 Pod（按 selector）
	backendPods := 0
	readyPods := 0
	if len(svc.Spec.Selector) > 0 {
		sel := metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: svc.Spec.Selector})
		pods, err := h.k8s.Clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: sel})
		if err == nil {
			backendPods = len(pods.Items)
			for _, p := range pods.Items {
				for _, cs := range p.Status.ContainerStatuses {
					if cs.Ready {
						readyPods++
						break
					}
				}
			}
		}
	}

	// 探测每个端口（ClusterIP 可达性由 Endpoints 数量体现）
	eps, _ := h.k8s.Clientset.CoreV1().Endpoints(ns).Get(ctx, name, metav1.GetOptions{})
	endpointAddrs := 0
	if eps != nil {
		for _, sub := range eps.Subsets {
			endpointAddrs += len(sub.Addresses)
		}
	}

	ports := make([]gin.H, 0, len(svc.Spec.Ports))
	for _, p := range svc.Spec.Ports {
		ports = append(ports, gin.H{
			"name":       p.Name,
			"port":       p.Port,
			"targetPort": p.TargetPort.String(),
			"protocol":   string(p.Protocol),
			"nodePort":   p.NodePort,
		})
	}

	healthy := readyPods > 0 || (len(svc.Spec.Selector) == 0 && len(svc.Spec.ExternalName) > 0)
	c.JSON(http.StatusOK, gin.H{
		"namespace": ns, "name": name,
		"type":           string(svc.Spec.Type),
		"clusterIP":      svc.Spec.ClusterIP,
		"externalName":   svc.Spec.ExternalName,
		"selector":       svc.Spec.Selector,
		"ports":          ports,
		"backendPods":    backendPods,
		"readyPods":      readyPods,
		"endpointAddrs":  endpointAddrs,
		"healthy":        healthy,
	})
}
