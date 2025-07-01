package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/valyala/fasthttp"
	"github.com/yourusername/k8s-controller-tutorial/pkg/informer"
	"k8s.io/client-go/util/homedir"
)

var (
	serverPort         int
	enableInformer     bool
	informerKubeconfig string
	informerInCluster  bool
	informerNamespace  string
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start a FastHTTP server with optional Kubernetes deployment informer",
	Long: `Start a FastHTTP server that can optionally run a Kubernetes deployment informer.
The informer watches for deployment events and logs them.

Examples:
  # Start server only
  k8s-controller server --port 8080

  # Start server with informer using kubeconfig
  k8s-controller server --port 8080 --enable-informer --kubeconfig ~/.kube/config

  # Start server with informer using in-cluster authentication
  k8s-controller server --port 8080 --enable-informer --in-cluster

  # Monitor specific namespace
  k8s-controller server --enable-informer --namespace production`,
	Run: func(cmd *cobra.Command, args []string) {
		level := parseLogLevel(logLevel)
		configureLogger(level)

		// Create context for graceful shutdown
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Set up signal handling
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		var wg sync.WaitGroup

		// Start HTTP server
		wg.Add(1)
		go func() {
			defer wg.Done()
			startHTTPServer(ctx)
		}()

		// Start informer if enabled
		if enableInformer {
			wg.Add(1)
			go func() {
				defer wg.Done()
				startDeploymentInformer(ctx)
			}()
		}

		// Wait for shutdown signal
		<-sigCh
		log.Info().Msg("Received shutdown signal, stopping services...")
		cancel()

		// Wait for all goroutines to finish
		wg.Wait()
		log.Info().Msg("All services stopped")
	},
}

func startHTTPServer(ctx context.Context) {
	handler := func(reqCtx *fasthttp.RequestCtx) {
		// Add deployment information if informer is enabled
		if enableInformer {
			fmt.Fprintf(reqCtx, "Hello from k8s-controller with deployment informer!\n")
			fmt.Fprintf(reqCtx, "Monitoring namespace: %s\n", informerNamespace)
			if informerInCluster {
				fmt.Fprintf(reqCtx, "Authentication: In-cluster\n")
			} else {
				fmt.Fprintf(reqCtx, "Authentication: Kubeconfig (%s)\n", getInformerKubeconfigPath())
			}
		} else {
			fmt.Fprintf(reqCtx, "Hello from k8s-controller!\n")
			fmt.Fprintf(reqCtx, "Informer: Disabled\n")
		}
		fmt.Fprintf(reqCtx, "Version: %s\n", appVersion)
	}

	addr := fmt.Sprintf(":%d", serverPort)
	log.Info().
		Int("port", serverPort).
		Bool("informer_enabled", enableInformer).
		Str("version", appVersion).
		Msg("Starting FastHTTP server")

	server := &fasthttp.Server{
		Handler: handler,
	}

	// Start server in background
	go func() {
		if err := server.ListenAndServe(addr); err != nil {
			log.Error().Err(err).Msg("FastHTTP server error")
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	log.Info().Msg("Shutting down HTTP server")

	// Graceful shutdown
	if err := server.Shutdown(); err != nil {
		log.Error().Err(err).Msg("Error during HTTP server shutdown")
	} else {
		log.Info().Msg("HTTP server stopped")
	}
}

func startDeploymentInformer(ctx context.Context) {
	log.Info().
		Str("namespace", informerNamespace).
		Bool("in_cluster", informerInCluster).
		Str("kubeconfig", getInformerKubeconfigPath()).
		Msg("Starting deployment informer")

	config := informer.InformerConfig{
		Kubeconfig: getInformerKubeconfigPath(),
		InCluster:  informerInCluster,
		Namespace:  informerNamespace,
	}

	deploymentInformer, err := informer.NewDeploymentInformer(config)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create deployment informer")
		return
	}

	// Start the informer
	if err := deploymentInformer.StartDeploymentInformer(ctx); err != nil {
		if err == context.Canceled {
			log.Info().Msg("Deployment informer stopped")
		} else {
			log.Error().Err(err).Msg("Deployment informer failed")
		}
	}
}

func getInformerKubeconfigPath() string {
	if !informerInCluster {
		if informerKubeconfig != "" {
			return informerKubeconfig
		}
		// Try environment variable
		if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
			return kubeconfig
		}
		// Default to ~/.kube/config
		if home := homedir.HomeDir(); home != "" {
			return filepath.Join(home, ".kube", "config")
		}
	}
	return ""
}

func init() {
	rootCmd.AddCommand(serverCmd)

	// Server flags
	serverCmd.Flags().IntVar(&serverPort, "port", 8080, "Port to run the HTTP server on")

	// Informer flags
	serverCmd.Flags().BoolVar(&enableInformer, "enable-informer", false, "Enable Kubernetes deployment informer")
	serverCmd.Flags().StringVar(&informerKubeconfig, "kubeconfig", "", "Path to kubeconfig file for informer (default: $HOME/.kube/config)")
	serverCmd.Flags().BoolVar(&informerInCluster, "in-cluster", false, "Use in-cluster authentication for informer")
	serverCmd.Flags().StringVar(&informerNamespace, "namespace", "default", "Kubernetes namespace to monitor")

	// Mark flags as mutually exclusive
	serverCmd.MarkFlagsMutuallyExclusive("kubeconfig", "in-cluster")
}
