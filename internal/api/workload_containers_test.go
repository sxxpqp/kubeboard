package api

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// 测试 summarizeContainerState（生产实现在 workload_containers.go）的三态映射。

func TestContainerStateMapping(t *testing.T) {
	cases := []struct {
		name      string
		status    corev1.ContainerStatus
		wantState string
		wantReason string
		wantReady bool
	}{
		{
			name: "running container",
			status: corev1.ContainerStatus{
				Name: "app", Ready: true, RestartCount: 2,
				State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: metav1.Now()}},
			},
			wantState: "Running", wantReady: true,
		},
		{
			name: "waiting crashloop",
			status: corev1.ContainerStatus{
				Name: "app", Ready: false, RestartCount: 7,
				State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
			},
			wantState: "Waiting", wantReason: "CrashLoopBackOff",
		},
		{
			name: "terminated with exit code",
			status: corev1.ContainerStatus{
				Name: "app", Ready: false, RestartCount: 1,
				State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Error", ExitCode: 1}},
			},
			wantState: "Terminated", wantReason: "Error",
		},
		{
			name:      "no state reported",
			status:    corev1.ContainerStatus{Name: "app"},
			wantState: "Unknown",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state, reason, _, ready := summarizeContainerState(tc.status)
			if state != tc.wantState {
				t.Errorf("state = %q, want %q", state, tc.wantState)
			}
			if reason != tc.wantReason {
				t.Errorf("reason = %q, want %q", reason, tc.wantReason)
			}
			if ready != tc.wantReady {
				t.Errorf("ready = %v, want %v", ready, tc.wantReady)
			}
		})
	}
}
