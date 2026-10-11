package api

import (
	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func derefInt32(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}

// listServiceAccounts 服务账号
func (h *Handler) listServiceAccounts(c *gin.Context) {
	ns := c.Query("namespace")
	list, err := h.k8s.Clientset.CoreV1().ServiceAccounts(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(list.Items))
	for _, sa := range list.Items {
		secrets := make([]string, 0, len(sa.Secrets))
		for _, s := range sa.Secrets {
			secrets = append(secrets, s.Name)
		}
		items = append(items, gin.H{
			"namespace": sa.Namespace, "name": sa.Name,
			"secrets":      secrets,
			"creationTime": sa.CreationTimestamp.Unix(),
		})
	}
	c.JSON(200, gin.H{"items": items, "total": len(items)})
}

// listPVC 持久卷声明
func (h *Handler) listPVC(c *gin.Context) {
	ns := c.Query("namespace")
	list, err := h.k8s.Clientset.CoreV1().PersistentVolumeClaims(ns).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(list.Items))
	for _, p := range list.Items {
		items = append(items, gin.H{
			"namespace": p.Namespace, "name": p.Name,
			"status":       string(p.Status.Phase),
			"capacity":     p.Status.Capacity.Storage().String(),
			"accessModes":  accessModes(p.Spec.AccessModes),
			"storageClass": derefStr(p.Spec.StorageClassName),
			"volumeName":   p.Spec.VolumeName,
			"creationTime": p.CreationTimestamp.Unix(),
		})
	}
	c.JSON(200, gin.H{"items": items, "total": len(items)})
}

// listPV 持久卷（集群级）
func (h *Handler) listPV(c *gin.Context) {
	list, err := h.k8s.Clientset.CoreV1().PersistentVolumes().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(list.Items))
	for _, p := range list.Items {
		items = append(items, gin.H{
			"name":         p.Name,
			"status":       string(p.Status.Phase),
			"capacity":     p.Spec.Capacity.Storage().String(),
			"accessModes":  accessModes(p.Spec.AccessModes),
			"storageClass": p.Spec.StorageClassName,
			"claim":        claimRef(p.Spec.ClaimRef),
			"reclaimPolicy": derefReclaim(p.Spec.PersistentVolumeReclaimPolicy),
			"creationTime": p.CreationTimestamp.Unix(),
		})
	}
	c.JSON(200, gin.H{"items": items, "total": len(items)})
}

// listStorageClasses 存储类
func (h *Handler) listStorageClasses(c *gin.Context) {
	list, err := h.k8s.Clientset.StorageV1().StorageClasses().List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(httpStatus(err), gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(list.Items))
	for _, sc := range list.Items {
		items = append(items, gin.H{
			"name":          sc.Name,
			"provisioner":   sc.Provisioner,
			"reclaimPolicy": derefReclaimPtr(sc.ReclaimPolicy),
			"volumeBinding": derefBinding(sc.VolumeBindingMode),
			"isDefault":     isDefaultClass(sc.Annotations),
			"creationTime":  sc.CreationTimestamp.Unix(),
		})
	}
	c.JSON(200, gin.H{"items": items, "total": len(items)})
}

func accessModes(modes []corev1.PersistentVolumeAccessMode) []string {
	out := make([]string, 0, len(modes))
	for _, m := range modes {
		out = append(out, string(m))
	}
	return out
}

func derefStr(p *string) string {
	if p == nil {
		return "-"
	}
	return *p
}

func derefReclaim(p corev1.PersistentVolumeReclaimPolicy) string {
	return string(p)
}

func derefReclaimPtr(p *corev1.PersistentVolumeReclaimPolicy) string {
	if p == nil {
		return "Delete"
	}
	return string(*p)
}

func derefBinding(p *storagev1.VolumeBindingMode) string {
	if p == nil {
		return "Immediate"
	}
	return string(*p)
}

func isDefaultClass(ann map[string]string) bool {
	if ann == nil {
		return false
	}
	return ann["storageclass.kubernetes.io/is-default-class"] == "true" ||
		ann["storageclass.beta.kubernetes.io/is-default-class"] == "true"
}

func claimRef(ref *corev1.ObjectReference) string {
	if ref == nil {
		return "-"
	}
	return ref.Namespace + "/" + ref.Name
}
