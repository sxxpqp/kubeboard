package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// revisionRevision 从 ReplicaSet 注解中读取 Deployment 的修订号
const revisionAnnotation = "deployment.kubernetes.io/revision"

type revisionInfo struct {
	Revision  int64  `json:"revision"`
	Name      string `json:"name"`      // ReplicaSet 名
	Replicas  int32  `json:"replicas"`  // 期望副本
	Ready     int32  `json:"ready"`     // 就绪副本
	Images    []string `json:"images"`
	Current   bool   `json:"current"`   // 是否当前生效版本
	CreatedAt int64  `json:"createdTime"`
	ChangeCause string `json:"changeCause"`
}

// workloadRevisions 返回 Deployment 的版本历史（等价 kubectl rollout history）
func (h *Handler) workloadRevisions(c *gin.Context) {
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
		sel, err := metav1.LabelSelectorAsSelector(d.Spec.Selector)
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		rsList, err := h.k8s.Clientset.AppsV1().ReplicaSets(ns).List(ctx, metav1.ListOptions{LabelSelector: sel.String()})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		currentRev := int64(0)
		if v, ok := d.Annotations[revisionAnnotation]; ok {
			currentRev, _ = strconv.ParseInt(v, 10, 64)
		}
		revs := make([]revisionInfo, 0, len(rsList.Items))
		for _, rs := range rsList.Items {
			rev, _ := strconv.ParseInt(rs.Annotations[revisionAnnotation], 10, 64)
			if rev == 0 {
				continue
			}
			desired := int32(0)
			if rs.Spec.Replicas != nil {
				desired = *rs.Spec.Replicas
			}
			cause := rs.Annotations["kubernetes.io/change-cause"]
			revs = append(revs, revisionInfo{
				Revision: rev, Name: rs.Name, Replicas: desired, Ready: rs.Status.ReadyReplicas,
				Images: containerImages(rs.Spec.Template.Spec.Containers),
				Current: rev == currentRev, CreatedAt: rs.CreationTimestamp.Unix(), ChangeCause: cause,
			})
		}
		sort.Slice(revs, func(i, j int) bool { return revs[i].Revision > revs[j].Revision })
		c.JSON(http.StatusOK, gin.H{"items": revs, "total": len(revs), "current": currentRev})
	case "statefulset":
		sts, err := h.k8s.Clientset.AppsV1().StatefulSets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		// StatefulSet 的版本由 ControllerRevision 管理（不是 ReplicaSet）
		crList, err := h.k8s.Clientset.AppsV1().ControllerRevisions(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		// 当前生效版本 = Status.CurrentRevision 指向的 ControllerRevision
		currentRevName := sts.Status.CurrentRevision
		currentNum := int64(0)
		revs := make([]revisionInfo, 0)
		for _, cr := range crList.Items {
			if cr.OwnerReferences == nil {
				continue
			}
			owned := false
			for _, o := range cr.OwnerReferences {
				if o.Kind == "StatefulSet" && o.Name == name {
					owned = true
					break
				}
			}
			if !owned {
				continue
			}
			desired := int32(0)
			if cr.Data.Raw != nil {
				var tpl appsv1.StatefulSet
				if err := jsonUnmarshal(cr.Data.Raw, &tpl); err == nil {
					if tpl.Spec.Replicas != nil {
						desired = *tpl.Spec.Replicas
					}
				}
			}
			isCurrent := cr.Name == currentRevName
			revs = append(revs, revisionInfo{
				Revision: cr.Revision, Name: cr.Name, Replicas: desired,
				Images: revisionImages(cr.Data.Raw),
				Current: isCurrent, CreatedAt: cr.CreationTimestamp.Unix(),
			})
			if isCurrent {
				currentNum = cr.Revision
			}
		}
		sort.Slice(revs, func(i, j int) bool { return revs[i].Revision > revs[j].Revision })
		c.JSON(http.StatusOK, gin.H{"items": revs, "total": len(revs), "current": currentNum})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "rollback 仅支持 deployment / statefulset"})
	}
}

// rolloutUndo 回滚到指定修订版本
func (h *Handler) rolloutUndo(c *gin.Context) {
	var req struct {
		Kind      string `json:"kind"`
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
		Revision  int64  `json:"revision"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()

	if req.Kind != "deployment" && req.Kind != "statefulset" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rollback 仅支持 deployment / statefulset"})
		return
	}

	if req.Kind == "deployment" {
		// Deployment：从 ReplicaSet 注解中找目标版本
		d, err := h.k8s.Clientset.AppsV1().Deployments(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		sel, err := metav1.LabelSelectorAsSelector(d.Spec.Selector)
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		rsList, err := h.k8s.Clientset.AppsV1().ReplicaSets(req.Namespace).List(ctx, metav1.ListOptions{LabelSelector: sel.String()})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		var target *appsv1.ReplicaSet
		for i := range rsList.Items {
			rev, _ := strconv.ParseInt(rsList.Items[i].Annotations[revisionAnnotation], 10, 64)
			if rev == req.Revision {
				target = &rsList.Items[i]
				break
			}
		}
		if target == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "revision not found: " + strconv.FormatInt(req.Revision, 10)})
			return
		}
		d.Spec.Template = *target.Spec.Template.DeepCopy()
		if d.Annotations == nil {
			d.Annotations = map[string]string{}
		}
		d.Annotations["kubernetes.io/change-cause"] = "rollback to revision " + strconv.FormatInt(req.Revision, 10)
		if _, err := h.k8s.Clientset.AppsV1().Deployments(req.Namespace).Update(ctx, d, metav1.UpdateOptions{}); err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
	} else {
		// StatefulSet：从 ControllerRevision 中找目标版本
		crList, err := h.k8s.Clientset.AppsV1().ControllerRevisions(req.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		var target *appsv1.ControllerRevision
		for i := range crList.Items {
			cr := &crList.Items[i]
			if cr.Revision != req.Revision {
				continue
			}
			owned := false
			for _, o := range cr.OwnerReferences {
				if o.Kind == "StatefulSet" && o.Name == req.Name {
					owned = true
					break
				}
			}
			if owned {
				target = cr
				break
			}
		}
		if target == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "revision not found: " + strconv.FormatInt(req.Revision, 10)})
			return
		}
		// 从 ControllerRevision 的 Raw 中还原 Pod 模板
		var tpl appsv1.StatefulSet
		if err := jsonUnmarshal(target.Data.Raw, &tpl); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "解析 revision 数据失败: " + err.Error()})
			return
		}
		s, err := h.k8s.Clientset.AppsV1().StatefulSets(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		s.Spec.Template = *tpl.Spec.Template.DeepCopy()
		if s.Annotations == nil {
			s.Annotations = map[string]string{}
		}
		s.Annotations["kubernetes.io/change-cause"] = "rollback to revision " + strconv.FormatInt(req.Revision, 10)
		if _, err := h.k8s.Clientset.AppsV1().StatefulSets(req.Namespace).Update(ctx, s, metav1.UpdateOptions{}); err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "rolledBackTo": req.Revision})
}

// relatedResource 关联资源（用于详情页互相跳转）
// 支持 controller（Deployment→RS→Pod）、service selector、node 上的 pod、ingress→service 等
func (h *Handler) relatedResources(c *gin.Context) {
	kind := strings.ToLower(c.Query("kind"))
	ns := c.Query("namespace")
	name := c.Query("name")
	ctx := c.Request.Context()

	out := []gin.H{}
	add := func(kind, ns, name, extra string) {
		out = append(out, gin.H{"kind": kind, "namespace": ns, "name": name, "extra": extra})
	}

	switch kind {
	case "pod":
		p, err := h.k8s.Clientset.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		// 节点
		if p.Spec.NodeName != "" {
			add("node", "", p.Spec.NodeName, "")
		}
		// owner（ReplicaSet / StatefulSet / Job 等）
		for _, o := range p.OwnerReferences {
			add(strings.ToLower(o.Kind), ns, o.Name, "owner")
		}
		// Service（按 selector 匹配）
		svcs, _ := h.k8s.Clientset.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
		for _, s := range svcs.Items {
			if len(s.Spec.Selector) == 0 {
				continue
			}
			matched := true
			for k, v := range s.Spec.Selector {
				if p.Labels[k] != v {
					matched = false
					break
				}
			}
			if matched {
				add("service", ns, s.Name, "selector")
			}
		}
		// ConfigMap / Secret（挂载或引用）
		for _, vol := range p.Spec.Volumes {
			if vol.ConfigMap != nil {
				add("configmap", ns, vol.ConfigMap.Name, "volume")
			}
			if vol.Secret != nil {
				add("secret", ns, vol.Secret.SecretName, "volume")
			}
			if vol.PersistentVolumeClaim != nil {
				add("pvc", ns, vol.PersistentVolumeClaim.ClaimName, "volume")
			}
		}
	case "deployment", "statefulset", "daemonset":
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
		}
		sel, err := metav1.LabelSelectorAsSelector(selector)
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		// Pods
		pods, _ := h.k8s.Clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: sel.String()})
		for _, p := range pods.Items {
			add("pod", ns, p.Name, "")
		}
		// ReplicaSets（Deployment 才有意义）
		if kind == "deployment" {
			rss, _ := h.k8s.Clientset.AppsV1().ReplicaSets(ns).List(ctx, metav1.ListOptions{LabelSelector: sel.String()})
			for _, rs := range rss.Items {
				rev := rs.Annotations[revisionAnnotation]
				add("replicaset", ns, rs.Name, "revision "+rev)
			}
		}
		// Services：用工作负载 Pod 模板标签匹配
		var tmplLabels map[string]string
		switch kind {
		case "deployment":
			d, _ := h.k8s.Clientset.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
			if d != nil {
				tmplLabels = d.Spec.Template.Labels
			}
		case "statefulset":
			st, _ := h.k8s.Clientset.AppsV1().StatefulSets(ns).Get(ctx, name, metav1.GetOptions{})
			if st != nil {
				tmplLabels = st.Spec.Template.Labels
			}
		case "daemonset":
			ds, _ := h.k8s.Clientset.AppsV1().DaemonSets(ns).Get(ctx, name, metav1.GetOptions{})
			if ds != nil {
				tmplLabels = ds.Spec.Template.Labels
			}
		}
		svcs, _ := h.k8s.Clientset.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
		for _, s := range svcs.Items {
			if len(s.Spec.Selector) == 0 {
				continue
			}
			matched := true
			for k, v := range s.Spec.Selector {
				if tmplLabels[k] != v {
					matched = false
					break
				}
			}
			if matched {
				add("service", ns, s.Name, "selector")
			}
		}
	case "service":
		s, err := h.k8s.Clientset.CoreV1().Services(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		if len(s.Spec.Selector) > 0 {
			sel := metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: s.Spec.Selector})
			pods, _ := h.k8s.Clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: sel})
			for _, p := range pods.Items {
				add("pod", ns, p.Name, "backend")
			}
		}
		// 引用了该 Service 的 Ingress
		ings, _ := h.k8s.Clientset.NetworkingV1().Ingresses(ns).List(ctx, metav1.ListOptions{})
		for _, ing := range ings.Items {
			for _, rule := range ing.Spec.Rules {
				if rule.HTTP == nil {
					continue
				}
				for _, path := range rule.HTTP.Paths {
					if path.Backend.Service != nil && path.Backend.Service.Name == name {
						add("ingress", ns, ing.Name, "backend")
					}
				}
			}
		}
	case "ingress":
		ing, err := h.k8s.Clientset.NetworkingV1().Ingresses(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		for _, rule := range ing.Spec.Rules {
			if rule.HTTP == nil {
				continue
			}
			for _, path := range rule.HTTP.Paths {
				if path.Backend.Service != nil {
					add("service", ns, path.Backend.Service.Name, "backend")
				}
			}
		}
		if ing.Spec.DefaultBackend != nil && ing.Spec.DefaultBackend.Service != nil {
			add("service", ns, ing.Spec.DefaultBackend.Service.Name, "default-backend")
		}
	case "node":
		pods, err := h.k8s.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{FieldSelector: "spec.nodeName=" + name})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		for _, p := range pods.Items {
			add("pod", p.Namespace, p.Name, "")
		}
	case "pvc":
		pvc, err := h.k8s.Clientset.CoreV1().PersistentVolumeClaims(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			c.JSON(httpStatus(err), gin.H{"error": err.Error()})
			return
		}
		if pvc.Spec.VolumeName != "" {
			add("pv", "", pvc.Spec.VolumeName, "bound")
		}
	default:
		c.JSON(http.StatusOK, gin.H{"items": out, "total": 0})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": out, "total": len(out)})
}

var _ = corev1.Pod{}
var _ = time.Now

// revisionImages 从 ControllerRevision 的 Raw 数据里提取容器镜像
func revisionImages(raw []byte) []string {
	if raw == nil {
		return nil
	}
	var probe struct {
		Spec struct {
			Template struct {
				Spec struct {
					Containers []struct {
						Image string `json:"image"`
					} `json:"containers"`
				} `json:"spec"`
			} `json:"template"`
		} `json:"spec"`
	}
	if err := jsonUnmarshal(raw, &probe); err != nil {
		return nil
	}
	out := make([]string, 0, len(probe.Spec.Template.Spec.Containers))
	for _, c := range probe.Spec.Template.Spec.Containers {
		out = append(out, c.Image)
	}
	return out
}

func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// relatedRef 关联资源引用（含 UID，供事件绑定复用）
type relatedRef struct {
	Kind, Namespace, Name, UID string
}

// collectRelated 收集与某资源关联的下游/上游资源（供 /api/related 与 /api/events/for 复用）
func (h *Handler) collectRelated(ctx context.Context, kind, ns, name string) []relatedRef {
	out := []relatedRef{}
	add := func(k, n, uid string) {
		out = append(out, relatedRef{Kind: k, Namespace: ns, Name: n, UID: uid})
	}

	switch kind {
	case "deployment", "statefulset", "daemonset":
		var selector *metav1.LabelSelector
		switch kind {
		case "deployment":
			if d, err := h.k8s.Clientset.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{}); err == nil {
				selector = d.Spec.Selector
			}
		case "statefulset":
			if st, err := h.k8s.Clientset.AppsV1().StatefulSets(ns).Get(ctx, name, metav1.GetOptions{}); err == nil {
				selector = st.Spec.Selector
			}
		case "daemonset":
			if ds, err := h.k8s.Clientset.AppsV1().DaemonSets(ns).Get(ctx, name, metav1.GetOptions{}); err == nil {
				selector = ds.Spec.Selector
			}
		}
		if selector == nil {
			return out
		}
		sel, err := metav1.LabelSelectorAsSelector(selector)
		if err != nil {
			return out
		}
		// 下游 Pod
		if pods, err := h.k8s.Clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: sel.String()}); err == nil {
			for _, p := range pods.Items {
				add("Pod", p.Name, string(p.UID))
			}
		}
		// 中间 ReplicaSet（Deployment）
		if kind == "deployment" {
			if rss, err := h.k8s.Clientset.AppsV1().ReplicaSets(ns).List(ctx, metav1.ListOptions{LabelSelector: sel.String()}); err == nil {
				for _, rs := range rss.Items {
					add("ReplicaSet", rs.Name, string(rs.UID))
				}
			}
		}
	case "service":
		if svc, err := h.k8s.Clientset.CoreV1().Services(ns).Get(ctx, name, metav1.GetOptions{}); err == nil {
			if len(svc.Spec.Selector) > 0 {
				sel := metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: svc.Spec.Selector})
				if pods, err := h.k8s.Clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: sel}); err == nil {
					for _, p := range pods.Items {
						add("Pod", p.Name, string(p.UID))
					}
				}
			}
		}
	case "node":
		if pods, err := h.k8s.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{FieldSelector: "spec.nodeName=" + name}); err == nil {
			for _, p := range pods.Items {
				add("Pod", p.Name, string(p.UID))
			}
		}
	}
	return out
}
