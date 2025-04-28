package pkg

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
)

type demoPlugin struct {
}

func NewDemoPlugin(_ runtime.Unknown, _ framework.FrameworkHandle) (framework.Plugin, error) {
	return &demoPlugin{}, nil
}

// Name returns name of the plugin. It is used in logs, etc.
func (d *demoPlugin) Name() string {
	return "demo-plugin"
}
func (d *demoPlugin) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	return 100, nil
}

// ScoreExtensions of the Score plugin.
func (d *demoPlugin) ScoreExtensions() framework.ScoreExtensions {
	return nil
}
