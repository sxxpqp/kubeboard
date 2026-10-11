package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (h *Handler) listIngresses(c *gin.Context) {
	ns := c.Query("namespace")
	list, err := h.k8s.Clientset.NetworkingV1().Ingresses(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(list.Items))
	for _, ing := range list.Items {
		items = append(items, gin.H{
			"namespace":    ing.Namespace,
			"name":         ing.Name,
			"className":    className(ing),
			"hosts":        ingressHosts(ing),
			"rules":        ingressRules(ing),
			"tls":          len(ing.Spec.TLS) > 0,
			"creationTime": ing.CreationTimestamp.Unix(),
		})
	}
	c.JSON(200, gin.H{"items": items, "total": len(items)})
}

func className(ing networkingv1.Ingress) string {
	if ing.Spec.IngressClassName != nil {
		return *ing.Spec.IngressClassName
	}
	if v, ok := ing.Annotations["kubernetes.io/ingress.class"]; ok {
		return v
	}
	return "-"
}

func ingressHosts(ing networkingv1.Ingress) []string {
	hosts := []string{}
	for _, r := range ing.Spec.Rules {
		if r.Host != "" {
			hosts = append(hosts, r.Host)
		}
	}
	if len(hosts) == 0 {
		hosts = append(hosts, "*")
	}
	return hosts
}

type ingressRule struct {
	Host  string `json:"host"`
	Path  string `json:"path"`
	Port  string `json:"port"`
	Service string `json:"service"`
}

func ingressRules(ing networkingv1.Ingress) []ingressRule {
	rules := []ingressRule{}
	for _, r := range ing.Spec.Rules {
		if r.HTTP == nil {
			continue
		}
		for _, p := range r.HTTP.Paths {
			svc := ""
			port := ""
			if p.Backend.Service != nil {
				svc = p.Backend.Service.Name
				if p.Backend.Service.Port.Number != 0 {
					port = fmt.Sprintf("%d", p.Backend.Service.Port.Number)
				} else {
					port = p.Backend.Service.Port.Name
				}
			}
			path := p.Path
			if path == "" {
				path = "/"
			}
			rules = append(rules, ingressRule{Host: r.Host, Path: path, Service: svc, Port: port})
		}
	}
	return rules
}
