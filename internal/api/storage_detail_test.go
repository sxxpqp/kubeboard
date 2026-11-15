package api

import "testing"

func TestPersistentVolumeDetailsKeepPVFields(t *testing.T) {
	spec, status := storageSpecStatus(map[string]interface{}{
		"kind": "PersistentVolume",
		"spec": map[string]interface{}{
			"capacity":         map[string]interface{}{"storage": "10Gi"},
			"storageClassName": "local", "persistentVolumeReclaimPolicy": "Retain",
			"claimRef": map[string]interface{}{"name": "data", "namespace": "default"},
		},
		"status": map[string]interface{}{"phase": "Bound"},
	})
	if spec["storageClass"] != "local" || status["phase"] != "Bound" || spec["claimRef"] == nil {
		t.Fatalf("PV fields lost: spec=%v status=%v", spec, status)
	}
	if _, exists := spec["volumeName"]; exists {
		t.Fatal("PV must not use PVC volumeName")
	}
}
