package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/schnell3526/k8s-job-notify/pkg/config"
	"github.com/schnell3526/k8s-job-notify/pkg/informer"
	"github.com/schnell3526/k8s-job-notify/pkg/notifier"
)

func main() {
	if err := run(); err != nil {
		slog.Error("failed to run", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	slog.Info("loaded configuration",
		"namespace", namespaceLogValue(cfg.Namespace),
		"in_cluster", cfg.InCluster,
		"resync_period", cfg.ResyncPeriod,
		"notification_level", cfg.NotificationLevel,
	)

	k8sConfig, err := buildK8sConfig(cfg.InCluster)
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return err
	}

	slackNotifier := notifier.NewSlackNotifier(cfg.SlackWebhookURL)
	jobInformer := informer.NewJobInformer(
		clientset,
		slackNotifier,
		cfg.Namespace,
		cfg.ResyncPeriod,
		cfg.NotificationLevel,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		slog.Info("received signal, shutting down", "signal", sig)
		cancel()
	}()

	return jobInformer.Run(ctx)
}

func buildK8sConfig(inCluster bool) (*rest.Config, error) {
	if inCluster {
		return rest.InClusterConfig()
	}

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		kubeconfig = home + "/.kube/config"
	}

	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

func namespaceLogValue(namespace string) string {
	if namespace == "" {
		return "all"
	}
	return namespace
}
