package k8s

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

type Client struct {
	Clientset    *kubernetes.Clientset
	Metrics      *metrics.Clientset
	Discovery    discovery.DiscoveryInterface
	RestConfig   *rest.Config
	Version      string
}

func New(kubeconfig string) (*Client, error) {
	var cfg *rest.Config
	var err error
	if kubeconfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else if icfg, icErr := rest.InClusterConfig(); icErr == nil {
		cfg = icfg
	} else {
		// 本地调试：回退到默认 kubeconfig (~/.kube/config)
		cfg, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(),
			&clientcmd.ConfigOverrides{}).ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("no kubeconfig found: set --kubeconfig or ensure ~/.kube/config exists: %w", err)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("build kube config: %w", err)
	}

	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create clientset: %w", err)
	}

	mc, err := metrics.NewForConfig(cfg)
	if err != nil {
		// metrics-server 可能未安装，降级为可用但无指标
		mc = nil
	}

	disc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create discovery client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	version := "unknown"
	if v, err := disc.ServerVersion(); err == nil {
		version = v.GitVersion
	}
	_ = ctx

	return &Client{
		Clientset:  cs,
		Metrics:    mc,
		Discovery:  disc,
		RestConfig: cfg,
		Version:    version,
	}, nil
}

// ClusterOverview 聚合集群级概览信息，对应 KubeSphere 首页。
type ClusterOverview struct {
	Version     string      `json:"version"`
	Nodes       NodeSummary `json:"nodes"`
	Namespaces  int64       `json:"namespaces"`
	Pods        PodSummary  `json:"pods"`
	Deployments int64       `json:"deployments"`
	Services    int64       `json:"services"`
	Events      int64       `json:"recentEvents"`
}

type NodeSummary struct {
	Total   int `json:"total"`
	Ready   int `json:"ready"`
}

type PodSummary struct {
	Total     int `json:"total"`
	Running   int `json:"running"`
	Pending   int `json:"pending"`
	Failed    int `json:"failed"`
	Succeeded int `json:"succeeded"`
}

func (c *Client) ClusterOverview(ctx context.Context) (*ClusterOverview, error) {
	ov := &ClusterOverview{Version: c.Version}

	nodes, err := c.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	ov.Nodes.Total = len(nodes.Items)
	for _, n := range nodes.Items {
		for _, cond := range n.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				ov.Nodes.Ready++
				break
			}
		}
	}

	nsList, err := c.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	ov.Namespaces = int64(len(nsList.Items))

	pods, err := c.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	ov.Pods.Total = len(pods.Items)
	for _, p := range pods.Items {
		switch p.Status.Phase {
		case corev1.PodRunning:
			ov.Pods.Running++
		case corev1.PodPending:
			ov.Pods.Pending++
		case corev1.PodFailed:
			ov.Pods.Failed++
		case corev1.PodSucceeded:
			ov.Pods.Succeeded++
		}
	}

	deploys, err := c.Clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	ov.Deployments = int64(len(deploys.Items))

	svcs, err := c.Clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	ov.Services = int64(len(svcs.Items))

	events, err := c.Clientset.CoreV1().Events("").List(ctx, metav1.ListOptions{Limit: 500})
	if err != nil {
		return nil, err
	}
	ov.Events = int64(len(events.Items))

	return ov, nil
}
