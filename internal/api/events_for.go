package api

import (
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// eventsFor 返回与指定资源绑定的事件
// GET /api/events/for?kind=&namespace=&name=&includeRelated=1
// 关联方式：优先 involvedObject.uid 精确匹配；includeRelated 时聚合下游资源事件
func (h *Handler) eventsFor(c *gin.Context) {
	kind := strings.ToLower(c.Query("kind"))
	ns := c.Query("namespace")
	name := c.Query("name")
	includeRelated := c.Query("includeRelated") == "1"
	if kind == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "kind and name required"})
		return
	}
	ctx := c.Request.Context()

	// 收集需要匹配的目标集合（uid 或 kind/name）
	type target struct{ kind, name, uid string }
	targets := map[string]target{} // key: uid 或 kind/name
	addTarget := func(k, n, uid string) {
		key := uid
		if key == "" {
			key = strings.ToLower(k) + "/" + n
		}
		targets[key] = target{kind: k, name: n, uid: uid}
	}

	// 1) 本资源
	var selfUID string
	if gvk, err := gvkFor(kind); err == nil {
		if ri, err := h.resourceClient(ctx, gvk, ns); err == nil {
			if obj, err := ri.Get(ctx, name, metav1.GetOptions{}); err == nil {
				selfUID = string(obj.GetUID())
				addTarget(gvk.Kind, name, selfUID)
			}
		}
	}
	if selfUID == "" {
		addTarget(kind, name, "")
	}

	// 2) 关联资源（下游）：复用 relatedResources 逻辑范围
	if includeRelated {
		for _, rel := range h.collectRelated(ctx, kind, ns, name) {
			addTarget(rel.Kind, rel.Name, rel.UID)
		}
	}

	// 拉取事件（同 namespace；node 等集群级少量 ns 用 ""）
	list, err := h.k8s.Clientset.CoreV1().Events(ns).List(ctx, metav1.ListOptions{Limit: 500})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0)
	for _, e := range list.Items {
		evUID := string(e.InvolvedObject.UID)
		evKey := ""
		if evUID != "" {
			evKey = evUID
		} else {
			evKey = strings.ToLower(e.InvolvedObject.Kind) + "/" + e.InvolvedObject.Name
		}
		if _, ok := targets[evKey]; !ok {
			continue
		}
		items = append(items, gin.H{
			"namespace": e.Namespace, "type": e.Type, "reason": e.Reason,
			"message": e.Message,
			"kind":    e.InvolvedObject.Kind, "name": e.InvolvedObject.Name,
			"uid":     evUID, "count": e.Count, "lastSeen": e.LastTimestamp.Unix(),
			"firstSeen": e.FirstTimestamp.Unix(),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return toI64(items[i]["lastSeen"]) > toI64(items[j]["lastSeen"])
	})
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items), "matchedUIDs": len(targets)})
}

func toI64(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	}
	return 0
}
