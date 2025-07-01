package cmd

import "testing"

func TestGetKubeClient_InvalidPath(t *testing.T) {
	originalKubeconfig := kubeconfig
	defer func() { kubeconfig = originalKubeconfig }()

	kubeconfig = "/invalid/path"

	_, err := getKubeClient()
	if err == nil {
		t.Error("expected error for invalid kubeconfig path")
	}
}
