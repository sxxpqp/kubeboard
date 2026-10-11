package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

// resourceDetail 统一详情接口：返回完整对象 + 规范化字段
// GET /api/resources/detail?kind=&namespace=&name=
func (h *Handler) resourceDetail(c *gin.Context) {
	kindParam := c.Query("kind")
	ns := c.Query("namespace")
	name := c.Query("name")
	if kindParam == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "kind and name required"})
		return
	}

	gvk, err := gvkFor(kindParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	ri, err := h.resourceClient(ctx, gvk, ns)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	obj, err := ri.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}

	content := obj.UnstructuredContent()

	// 元数据（权威字段，不依赖列表）
	meta := gin.H{
		"uid":             metaString(content, "metadata", "uid"),
		"resourceVersion": metaString(content, "metadata", "resourceVersion"),
		"generation":      metaInt(content, "metadata", "generation"),
		"creationTime":    metaUnix(content, "metadata", "creationTimestamp"),
		"labels":          metaMap(content, "metadata", "labels"),
		"annotations":     metaMap(content, "metadata", "annotations"),
		"ownerReferences": metaSlice(content, "metadata", "ownerReferences"),
	}

	// 规格摘要（通用 + 类型专属）
	spec := gin.H{}
	status := gin.H{}
	switch strings.ToLower(kindParam) {
	case "pod":
		spec, status = podSpecStatus(content)
	case "deployment", "statefulset", "daemonset", "replicaset":
		spec, status = workloadSpecStatus(content)
	case "service":
		spec, status = serviceSpecStatus(content)
	case "configmap", "secret":
		spec = dataSpec(content)
	case "pvc", "pv":
		spec, status = storageSpecStatus(content)
	case "node":
		spec, status = nodeSpecStatus(content)
	case "ingress":
		spec, status = ingressSpecStatus(content)
	}

	// 完整 YAML
	yamlBytes, _ := yaml.Marshal(content)

	c.JSON(http.StatusOK, gin.H{
		"kind":        gvk.Kind,
		"apiVersion":  gvk.GroupVersion().String(),
		"namespace":   ns,
		"name":        name,
		"metadata":    meta,
		"spec":        spec,
		"status":      status,
		"rawYaml":     string(yamlBytes),
	})
}

func metaString(content map[string]interface{}, path ...string) string {
	v := dig(content, path...)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
func metaInt(content map[string]interface{}, path ...string) int64 {
	v := dig(content, path...)
	if f, ok := v.(float64); ok {
		return int64(f)
	}
	return 0
}
func metaUnix(content map[string]interface{}, path ...string) int64 {
	v := dig(content, path...)
	if s, ok := v.(string); ok && s != "" {
		if t, err := timeParse(s); err == nil {
			return t
		}
	}
	return 0
}
func metaMap(content map[string]interface{}, path ...string) map[string]string {
	v := dig(content, path...)
	m, ok := v.(map[string]interface{})
	if !ok {
		return map[string]string{}
	}
	out := make(map[string]string, len(m))
	for k, val := range m {
		if s, ok := val.(string); ok {
			out[k] = s
		}
	}
	return out
}
func metaSlice(content map[string]interface{}, path ...string) []interface{} {
	v := dig(content, path...)
	if s, ok := v.([]interface{}); ok {
		return s
	}
	return []interface{}{}
}

func dig(m map[string]interface{}, path ...string) interface{} {
	var cur interface{} = m
	for _, p := range path {
		mm, ok := cur.(map[string]interface{})
		if !ok {
			return nil
		}
		cur = mm[p]
	}
	return cur
}

// —— 类型专属 spec/status 提取 ——

func podSpecStatus(content map[string]interface{}) (gin.H, gin.H) {
	var p corev1.Pod
	b, _ := json.Marshal(content)
	_ = json.Unmarshal(b, &p)

	containers := make([]gin.H, 0, len(p.Spec.Containers))
	for _, c := range p.Spec.Containers {
		cs := gin.H{
			"name": c.Name, "image": c.Image,
			"imagePullPolicy": string(c.ImagePullPolicy),
			"command":         c.Command, "args": c.Args,
			"ports":           c.Ports, "env": envList(c.Env),
			"resources":       c.Resources,
			"readinessProbe":  c.ReadinessProbe, "livenessProbe": c.LivenessProbe,
		}
		containers = append(containers, cs)
	}
	volumes := make([]gin.H, 0, len(p.Spec.Volumes))
	for _, v := range p.Spec.Volumes {
		vi := gin.H{"name": v.Name}
		if v.ConfigMap != nil {
			vi["configMap"] = v.ConfigMap.Name
		}
		if v.Secret != nil {
			vi["secret"] = v.Secret.SecretName
		}
		if v.PersistentVolumeClaim != nil {
			vi["pvc"] = v.PersistentVolumeClaim.ClaimName
		}
		if v.HostPath != nil {
			vi["hostPath"] = v.HostPath.Path
		}
		if v.EmptyDir != nil {
			vi["emptyDir"] = true
		}
		volumes = append(volumes, vi)
	}
	conditions := make([]gin.H, 0, len(p.Status.Conditions))
	for _, c := range p.Status.Conditions {
		conditions = append(conditions, gin.H{
			"type": string(c.Type), "status": string(c.Status),
			"reason": c.Reason, "message": c.Message,
			"lastTransitionTime": c.LastTransitionTime.Unix(),
		})
	}
	cstatuses := make([]gin.H, 0, len(p.Status.ContainerStatuses))
	for _, cs := range p.Status.ContainerStatuses {
		e := gin.H{"name": cs.Name, "ready": cs.Ready, "restartCount": cs.RestartCount, "image": cs.Image}
		if cs.State.Running != nil {
			e["state"] = "Running"
			e["startedAt"] = cs.State.Running.StartedAt.Unix()
		} else if cs.State.Waiting != nil {
			e["state"] = "Waiting"
			e["reason"] = cs.State.Waiting.Reason
			e["message"] = cs.State.Waiting.Message
		} else if cs.State.Terminated != nil {
			e["state"] = "Terminated"
			e["reason"] = cs.State.Terminated.Reason
			e["exitCode"] = cs.State.Terminated.ExitCode
		}
		cstatuses = append(cstatuses, e)
	}
	spec := gin.H{
		"containers": containers, "volumes": volumes,
		"nodeName": p.Spec.NodeName, "serviceAccount": p.Spec.ServiceAccountName,
		"restartPolicy": string(p.Spec.RestartPolicy), "schedulerName": p.Spec.SchedulerName,
		"nodeSelector": p.Spec.NodeSelector, "tolerations": p.Spec.Tolerations,
	}
	status := gin.H{
		"phase": string(p.Status.Phase), "podIP": p.Status.PodIP, "hostIP": p.Status.HostIP,
		"qosClass": string(p.Status.QOSClass), "conditions": conditions,
		"containerStatuses": cstatuses, "startTime": unixOrZero(p.Status.StartTime),
		"message": p.Status.Message, "reason": p.Status.Reason,
	}
	return spec, status
}

func envList(envs []corev1.EnvVar) []gin.H {
	out := make([]gin.H, 0, len(envs))
	for _, e := range envs {
		item := gin.H{"name": e.Name, "value": e.Value}
		if e.ValueFrom != nil {
			if e.ValueFrom.ConfigMapKeyRef != nil {
				item["fromConfigMap"] = e.ValueFrom.ConfigMapKeyRef.Name + "/" + e.ValueFrom.ConfigMapKeyRef.Key
			}
			if e.ValueFrom.SecretKeyRef != nil {
				item["fromSecret"] = e.ValueFrom.SecretKeyRef.Name + "/" + e.ValueFrom.SecretKeyRef.Key
			}
		}
		out = append(out, item)
	}
	return out
}

func workloadSpecStatus(content map[string]interface{}) (gin.H, gin.H) {
	var d appsv1.Deployment
	b, _ := json.Marshal(content)
	_ = json.Unmarshal(b, &d)

	containers := make([]gin.H, 0, len(d.Spec.Template.Spec.Containers))
	for _, c := range d.Spec.Template.Spec.Containers {
		containers = append(containers, gin.H{
			"name": c.Name, "image": c.Image, "ports": c.Ports,
			"env": envList(c.Env), "resources": c.Resources,
		})
	}
	desired := int32(0)
	if d.Spec.Replicas != nil {
		desired = *d.Spec.Replicas
	}
	strategy := string(d.Spec.Strategy.Type)
	var maxSurge, maxUnavailable string
	if d.Spec.Strategy.RollingUpdate != nil {
		if d.Spec.Strategy.RollingUpdate.MaxSurge != nil {
			maxSurge = d.Spec.Strategy.RollingUpdate.MaxSurge.String()
		}
		if d.Spec.Strategy.RollingUpdate.MaxUnavailable != nil {
			maxUnavailable = d.Spec.Strategy.RollingUpdate.MaxUnavailable.String()
		}
	}
	selMap := map[string]string{}
	if d.Spec.Selector != nil {
		selMap = d.Spec.Selector.MatchLabels
	}
	spec := gin.H{
		"replicas": desired, "containers": containers,
		"selector": selMap, "strategy": strategy,
		"maxSurge": maxSurge, "maxUnavailable": maxUnavailable,
		"paused": d.Spec.Paused,
		"podLabels": d.Spec.Template.Labels,
		"volumes": d.Spec.Template.Spec.Volumes,
	}
	status := gin.H{
		"replicas": d.Status.Replicas, "readyReplicas": d.Status.ReadyReplicas,
		"availableReplicas": d.Status.AvailableReplicas, "updatedReplicas": d.Status.UpdatedReplicas,
		"unavailableReplicas": d.Status.UnavailableReplicas,
		"observation":         gin.H{"observedGeneration": d.Status.ObservedGeneration, "generation": d.Generation},
	}
	return spec, status
}

func serviceSpecStatus(content map[string]interface{}) (gin.H, gin.H) {
	var s corev1.Service
	b, _ := json.Marshal(content)
	_ = json.Unmarshal(b, &s)
	ports := make([]gin.H, 0, len(s.Spec.Ports))
	for _, p := range s.Spec.Ports {
		ports = append(ports, gin.H{
			"name": p.Name, "port": p.Port, "targetPort": p.TargetPort.String(),
			"protocol": string(p.Protocol), "nodePort": p.NodePort,
		})
	}
	spec := gin.H{
		"type": string(s.Spec.Type), "clusterIP": s.Spec.ClusterIP,
		"externalName": s.Spec.ExternalName, "selector": s.Spec.Selector,
		"ports": ports, "sessionAffinity": string(s.Spec.SessionAffinity),
	}
	status := gin.H{}
	if s.Spec.Type == corev1.ServiceTypeLoadBalancer {
		status["loadBalancer"] = s.Status.LoadBalancer
	}
	return spec, status
}

func dataSpec(content map[string]interface{}) gin.H {
	data := gin.H{}
	if m, ok := content["data"].(map[string]interface{}); ok {
		for k, v := range m {
			if s, ok := v.(string); ok {
				if len(s) > 500 {
					s = s[:500] + "…(已截断)"
				}
				data[k] = s
			}
		}
	}
	if m, ok := content["stringData"].(map[string]interface{}); ok {
		data["__stringData"] = m
	}
	return gin.H{"data": data, "keys": strKeys(data)}
}

func strKeys(m gin.H) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func storageSpecStatus(content map[string]interface{}) (gin.H, gin.H) {
	var pvc corev1.PersistentVolumeClaim
	b, _ := json.Marshal(content)
	_ = json.Unmarshal(b, &pvc)
	spec := gin.H{
		"accessModes": pvc.Spec.AccessModes, "volumeName": pvc.Spec.VolumeName,
		"storageClass": strOrEmpty(pvc.Spec.StorageClassName),
		"capacity":     bytesOrPct(pvc.Spec.Resources.Requests),
	}
	status := gin.H{"phase": string(pvc.Status.Phase), "capacity": pvc.Status.Capacity}
	return spec, status
}

func nodeSpecStatus(content map[string]interface{}) (gin.H, gin.H) {
	var n corev1.Node
	b, _ := json.Marshal(content)
	_ = json.Unmarshal(b, &n)
	conditions := make([]gin.H, 0, len(n.Status.Conditions))
	for _, cond := range n.Status.Conditions {
		conditions = append(conditions, gin.H{
			"type": string(cond.Type), "status": string(cond.Status),
			"reason": cond.Reason, "message": cond.Message,
		})
	}
	spec := gin.H{
		"unschedulable": n.Spec.Unschedulable, "podCIDR": n.Spec.PodCIDR,
		"taints": n.Spec.Taints, "providerID": n.Spec.ProviderID,
	}
	status := gin.H{
		"capacity": n.Status.Capacity, "allocatable": n.Status.Allocatable,
		"conditions": conditions, "nodeInfo": n.Status.NodeInfo,
		"addresses": n.Status.Addresses,
	}
	return spec, status
}

func ingressSpecStatus(content map[string]interface{}) (gin.H, gin.H) {
	spec := gin.H{}
	rules := []gin.H{}
	if rs, ok := content["spec"].(map[string]interface{}); ok {
		if rl, ok := rs["rules"].([]interface{}); ok {
			for _, r := range rl {
				rm, _ := r.(map[string]interface{})
				host, _ := rm["host"].(string)
				paths := []gin.H{}
				if httpm, ok := rm["http"].(map[string]interface{}); ok {
					if pl, ok := httpm["paths"].([]interface{}); ok {
						for _, p := range pl {
							pm, _ := p.(map[string]interface{})
							entry := gin.H{"path": pm["path"], "pathType": pm["pathType"]}
							if bm, ok := pm["backend"].(map[string]interface{}); ok {
								if sm, ok := bm["service"].(map[string]interface{}); ok {
									entry["service"] = sm["name"]
									if portm, ok := sm["port"].(map[string]interface{}); ok {
										entry["port"] = portm["number"]
										if entry["port"] == nil {
											entry["port"] = portm["name"]
										}
									}
								}
							}
							paths = append(paths, entry)
						}
					}
				}
				rules = append(rules, gin.H{"host": host, "paths": paths})
			}
		}
		if ic, ok := rs["ingressClassName"].(string); ok {
			spec["ingressClassName"] = ic
		}
		if tls, ok := rs["tls"].([]interface{}); ok {
			spec["tls"] = tls
		}
	}
	spec["rules"] = rules
	return spec, gin.H{}
}

var _ = schema.GroupVersionKind{}

func timeParse(s string) (int64, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
}

func unixOrZero(t *metav1.Time) int64 {
	if t == nil {
		return 0
	}
	return t.Unix()
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func bytesOrPct(rl corev1.ResourceList) string {
	for k, v := range rl {
		if k == corev1.ResourceStorage {
			return v.String()
		}
	}
	return ""
}
