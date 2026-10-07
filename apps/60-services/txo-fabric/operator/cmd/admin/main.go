package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/admin"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "oauth-connect":
		if err := runOAuthConnect(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "ERROR:", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: txo-fabric-admin oauth-connect --tenant <name> [--provider codex] [--timeout 6m]")
}

func runOAuthConnect(args []string) error {
	flags := flag.NewFlagSet("oauth-connect", flag.ContinueOnError)
	tenant := flags.String("tenant", "", "TenantBundle metadata.name")
	provider := flags.String("provider", "codex", "OAuth provider (currently codex)")
	timeout := flags.Duration("timeout", 6*time.Minute, "maximum time to wait for OAuth completion")
	pollInterval := flags.Duration("poll-interval", 2*time.Second, "OAuth status polling interval")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*tenant) == "" {
		return fmt.Errorf("--tenant is required")
	}
	if strings.ToLower(strings.TrimSpace(*provider)) != "codex" {
		return fmt.Errorf("provider %q is not supported yet", *provider)
	}
	if *timeout <= 0 {
		return fmt.Errorf("--timeout must be positive")
	}
	if *pollInterval <= 0 {
		return fmt.Errorf("--poll-interval must be positive")
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("load in-cluster Kubernetes config: %w", err)
	}
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return fmt.Errorf("add Kubernetes scheme: %w", err)
	}
	if err := fabricv1alpha1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("add Fabric scheme: %w", err)
	}
	kubeClient, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("create Kubernetes client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	target, err := admin.ResolveTenantBroker(ctx, kubeClient, *tenant)
	if err != nil {
		return err
	}
	cpa := admin.NewCPAClient(target.BaseURL, target.ManagementCredential)

	start, err := cpa.StartCodexOAuth(ctx)
	if err != nil {
		return fmt.Errorf("start Codex OAuth for tenant %q: %w", target.TenantName, err)
	}

	fmt.Printf("TENANT=%s\n", target.TenantName)
	fmt.Printf("PROVIDER=codex\n")
	fmt.Printf("AUTHORIZATION_URL=%s\n", start.URL)
	fmt.Printf("OAUTH_STATE=%s\n", start.State)
	fmt.Println("STATUS=waiting")
	fmt.Println("Open AUTHORIZATION_URL in your browser and complete the OpenAI/Codex authorization.")
	fmt.Println("When the browser reaches the localhost callback URL, copy the FULL URL from the address bar and paste it below.")
	fmt.Print("CALLBACK_URL> ")

	reader := bufio.NewReader(os.Stdin)
	callbackURL, err := reader.ReadString('\n')
	if err != nil {
		cancelCtx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
		_ = cpa.CancelOAuth(cancelCtx, start.State)
		cancelFn()
		return fmt.Errorf("read OAuth callback URL: %w", err)
	}
	callbackURL = strings.TrimSpace(callbackURL)
	if callbackURL == "" {
		cancelCtx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
		_ = cpa.CancelOAuth(cancelCtx, start.State)
		cancelFn()
		return fmt.Errorf("OAuth callback URL is empty")
	}
	if err := cpa.SubmitOAuthCallback(ctx, "codex", start.State, callbackURL); err != nil {
		return fmt.Errorf("submit Codex OAuth callback: %w", err)
	}
	fmt.Println("CALLBACK_SUBMITTED")

	ticker := time.NewTicker(*pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			cancelCtx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
			_ = cpa.CancelOAuth(cancelCtx, start.State)
			cancelFn()
			return fmt.Errorf("OAuth authorization did not complete before timeout")
		case <-ticker.C:
			status, err := cpa.GetOAuthStatus(ctx, start.State)
			if err != nil {
				return fmt.Errorf("poll Codex OAuth status: %w", err)
			}
			switch status.Status {
			case "wait":
				continue
			case "ok":
				fmt.Println("STATUS=ok")
				fmt.Println("CODEX_OAUTH_CONNECTED")
				return nil
			case "error":
				if strings.TrimSpace(status.Error) == "" {
					return fmt.Errorf("Codex OAuth failed")
				}
				return fmt.Errorf("Codex OAuth failed: %s", status.Error)
			default:
				return fmt.Errorf("unexpected CPA OAuth status %q", status.Status)
			}
		}
	}
}
