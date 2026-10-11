package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	meta "k8s.io/apimachinery/pkg/api/meta"

	"github.com/sxxpqp/kubeboard/internal/auth"
	"github.com/sxxpqp/kubeboard/internal/k8s"
)

type Handler struct {
	auth       *auth.Manager
	k8s        *k8s.Client
	dyn        dynamic.Interface
	mapper     meta.RESTMapper
	enableExec bool
}

func New(a *auth.Manager, c *k8s.Client, enableExec bool) *Handler {
	return &Handler{auth: a, k8s: c, enableExec: enableExec}
}

func (h *Handler) Register(r *gin.Engine) {
	r.POST("/api/login", h.login)
	r.GET("/api/version", h.version)

	api := r.Group("/api")
	api.Use(h.jwtMiddleware())
	{
		api.GET("/overview", h.overview)
		api.GET("/nodes", h.listNodes)
		api.GET("/namespaces", h.listNamespaces)
		api.GET("/workloads", h.listWorkloads)
		api.GET("/services", h.listServices)
		api.GET("/configmaps", h.listConfigMaps)
		api.GET("/secrets", h.listSecrets)
		api.GET("/events", h.listEvents)
		api.GET("/pods", h.listPods)
		api.GET("/resource", h.getResource)
		api.POST("/resources", h.createResource)
		api.PUT("/resources", h.updateResource)
		api.DELETE("/resources", h.deleteResource)
		api.POST("/workloads/scale", h.scaleWorkload)
		api.POST("/workloads/image", h.updateImage)
		api.POST("/workloads/restart", h.rolloutRestart)
		api.GET("/workloads/pods", h.workloadPods)
		api.GET("/pods/logs", h.podLogs)
		if h.enableExec {
			api.GET("/pods/terminal", h.podTerminal)
		}
		api.GET("/ingresses", h.listIngresses)
		api.GET("/serviceaccounts", h.listServiceAccounts)
		api.GET("/pvcs", h.listPVC)
		api.GET("/pvs", h.listPV)
		api.GET("/storageclasses", h.listStorageClasses)
	}
}

func (h *Handler) jwtMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := ""
		if header := c.GetHeader("Authorization"); strings.HasPrefix(header, "Bearer ") {
			tokenStr = strings.TrimPrefix(header, "Bearer ")
		} else if q := c.Query("token"); q != "" {
			// WebSocket 无法设置请求头，允许通过查询参数传递令牌
			tokenStr = q
		}
		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		claims, err := h.auth.Verify(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("username", claims.Username)
		c.Next()
	}
}

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, err := h.auth.Login(req.Username, req.Password, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "username": req.Username})
}

func (h *Handler) version(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":    h.k8s.Version,
		"metrics":    h.k8s.Metrics != nil,
		"exec":       h.enableExec,
		"consoleVer": "v3.2.0",
	})
}

func (h *Handler) overview(c *gin.Context) {
	ov, err := h.k8s.ClusterOverview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ov)
}

func (h *Handler) listNodes(c *gin.Context) {
	ctx := c.Request.Context()
	nodes, err := h.k8s.Clientset.CoreV1().Nodes().List(ctx, nodeListOptions(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(nodes.Items))
	for _, n := range nodes.Items {
		ready := "Unknown"
		for _, cond := range n.Status.Conditions {
			if cond.Type == "Ready" {
				ready = string(cond.Status)
			}
		}
		items = append(items, gin.H{
			"name":        n.Name,
			"status":      ready,
			"roles":       nodeRoles(n.Labels),
			"version":     n.Status.NodeInfo.KubeletVersion,
			"os":          n.Status.NodeInfo.OSImage,
			"kernel":      n.Status.NodeInfo.KernelVersion,
			"internalIP":  nodeInternalIP(n.Status.Addresses),
			"creationTime": n.CreationTimestamp.Unix(),
			"cpu":         n.Status.Capacity.Cpu().String(),
			"memory":      n.Status.Capacity.Memory().String(),
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func nodeListOptions(c *gin.Context) metav1.ListOptions {
	return metav1.ListOptions{}
}
