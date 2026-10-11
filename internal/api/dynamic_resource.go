package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"sigs.k8s.io/yaml"
)

var groupVersionKinds = map[string]schema.GroupVersionKind{
	"deployment":  {Group: "apps", Version: "v1", Kind: "Deployment"},
	"statefulset": {Group: "apps", Version: "v1", Kind: "StatefulSet"},
	"daemonset":   {Group: "apps", Version: "v1", Kind: "DaemonSet"},
	"pod":         {Group: "", Version: "v1", Kind: "Pod"},
	"service":     {Group: "", Version: "v1", Kind: "Service"},
	"configmap":   {Group: "", Version: "v1", Kind: "ConfigMap"},
	"secret":      {Group: "", Version: "v1", Kind: "Secret"},
	"namespace":   {Group: "", Version: "v1", Kind: "Namespace"},
	"ingress":     {Group: "networking.k8s.io", Version: "v1", Kind: "Ingress"},
	"job":         {Group: "batch", Version: "v1", Kind: "Job"},
	"cronjob":     {Group: "batch", Version: "v1", Kind: "CronJob"},
	"serviceaccount": {Group: "", Version: "v1", Kind: "ServiceAccount"},
	"pvc":         {Group: "", Version: "v1", Kind: "PersistentVolumeClaim"},
	"pv":          {Group: "", Version: "v1", Kind: "PersistentVolume"},
	"storageclass": {Group: "storage.k8s.io", Version: "v1", Kind: "StorageClass"},
	"node":        {Group: "", Version: "v1", Kind: "Node"},
	"replicaset":  {Group: "apps", Version: "v1", Kind: "ReplicaSet"},
	"endpoints":   {Group: "", Version: "v1", Kind: "Endpoints"},
	"limitrange":  {Group: "", Version: "v1", Kind: "LimitRange"},
	"hpa":         {Group: "autoscaling", Version: "v2", Kind: "HorizontalPodAutoscaler"},
	"networkpolicy": {Group: "networking.k8s.io", Version: "v1", Kind: "NetworkPolicy"},
	"resourcequota": {Group: "", Version: "v1", Kind: "ResourceQuota"},
}

func (h *Handler) resourceClient(ctx context.Context, gvk schema.GroupVersionKind, namespace string) (dynamic.ResourceInterface, error) {
	if h.dyn == nil {
		dc, err := dynamic.NewForConfig(h.k8s.RestConfig)
		if err != nil {
			return nil, err
		}
		h.dyn = dc

		discoveryClient, err := discovery.NewDiscoveryClientForConfig(h.k8s.RestConfig)
		if err != nil {
			return nil, err
		}
		groupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
		if err != nil {
			return nil, err
		}
		h.mapper = restmapper.NewDiscoveryRESTMapper(groupResources)
	}
	mapping, err := h.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("resolve REST mapping: %w", err)
	}
	if mapping.Scope.Name() == metaScopeRoot || namespace == "" {
		return h.dyn.Resource(mapping.Resource), nil
	}
	return h.dyn.Resource(mapping.Resource).Namespace(namespace), nil
}

const metaScopeRoot = "root"

func gvkFor(kindParam string) (schema.GroupVersionKind, error) {
	gvk, ok := groupVersionKinds[strings.ToLower(kindParam)]
	if !ok {
		return schema.GroupVersionKind{}, fmt.Errorf("unsupported kind: %s", kindParam)
	}
	return gvk, nil
}

func (h *Handler) getResource(c *gin.Context) {
	kindParam := c.Query("kind")
	namespace := c.Query("namespace")
	name := c.Query("name")

	gvk, err := gvkFor(kindParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ri, err := h.resourceClient(c.Request.Context(), gvk, namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	obj, err := ri.Get(c.Request.Context(), name, metav1.GetOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	out, err := yaml.Marshal(obj.UnstructuredContent())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/x-yaml", out)
}

type yamlReq struct {
	Kind      string `json:"kind,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	YAML      string `json:"yaml" binding:"required"`
}

func (h *Handler) createResource(c *gin.Context) {
	var req yamlReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	obj, err := parseYAML(req.YAML)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	gvk := obj.GroupVersionKind()
	if gvk.Kind == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing kind in yaml"})
		return
	}
	ri, err := h.resourceClient(c.Request.Context(), gvk, obj.GetNamespace())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	created, err := ri.Create(c.Request.Context(), obj, metav1.CreateOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	out, _ := yaml.Marshal(created.UnstructuredContent())
	c.Data(http.StatusCreated, "application/x-yaml", out)
}

func (h *Handler) updateResource(c *gin.Context) {
	var req yamlReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	obj, err := parseYAML(req.YAML)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	gvk := obj.GroupVersionKind()
	if gvk.Kind == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing kind in yaml"})
		return
	}
	ri, err := h.resourceClient(c.Request.Context(), gvk, obj.GetNamespace())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	updated, err := ri.Update(c.Request.Context(), obj, metav1.UpdateOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	out, _ := yaml.Marshal(updated.UnstructuredContent())
	c.Data(http.StatusOK, "application/x-yaml", out)
}

func (h *Handler) deleteResource(c *gin.Context) {
	kindParam := c.Query("kind")
	namespace := c.Query("namespace")
	name := c.Query("name")

	gvk, err := gvkFor(kindParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ri, err := h.resourceClient(c.Request.Context(), gvk, namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := ri.Delete(c.Request.Context(), name, metav1.DeleteOptions{}); err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": name})
}

func parseYAML(data string) (*unstructured.Unstructured, error) {
	var m map[string]interface{}
	if err := yaml.Unmarshal([]byte(data), &m); err != nil {
		return nil, fmt.Errorf("invalid yaml: %w", err)
	}
	if m == nil {
		return nil, fmt.Errorf("empty yaml")
	}
	return &unstructured.Unstructured{Object: m}, nil
}
