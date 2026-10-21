package main

import (
	"flag"
	"os"
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/controller"
	"github.com/charchess/vixens/apps/60-services/txo-fabric/operator/internal/privatebridge"
	"k8s.io/client-go/kubernetes"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		panic(err)
	}
	if err := fabricv1alpha1.AddToScheme(scheme); err != nil {
		panic(err)
	}
}

func main() {
	var metricsAddr string
	var probeAddr string
	var enableLeaderElection bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", true, "Enable leader election for controller manager.")
	zapOpts := zap.Options{Development: false}
	zapOpts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOpts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "txo-fabric-operator.fabric.truxonline.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err := (&controller.TenantBundleReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create TenantBundle controller")
		os.Exit(1)
	}
	if err := (&controller.SharedWorkspaceReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create SharedWorkspace controller")
		os.Exit(1)
	}
	if err := (&controller.AgentIdentityReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create AgentIdentity controller")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Source-only contract: this listener is NOT reachable or started by
	// default. Activation requires a separate, reviewed platform-secret
	// bootstrap, BFF ServiceAccount deployment and GitOps gate change.
	if os.Getenv("TXO_FABRIC_PRIVATE_BRIDGE_ENABLED") == "true" {
		key:=[]byte(os.Getenv("TXO_FABRIC_PRIVATE_BRIDGE_HMAC_KEY"))
		issuer:=os.Getenv("TXO_FABRIC_PRIVATE_BRIDGE_K8S_ISSUER")
		const audience="txo-fabric-private-identity"
		const subject="system:serviceaccount:txo-fabric-system:txo-fabric-webui-bff"
		if len(key)<32 || issuer=="" {
			setupLog.Error(errors.New("private bridge configuration incomplete"),"refusing unauthenticated identity API")
			os.Exit(1)
		}
		kube,err:=kubernetes.NewForConfig(mgr.GetConfig())
		if err!=nil {
			setupLog.Error(err,"cannot initialize private TokenReview client")
			os.Exit(1)
		}
		handler:=&privatebridge.Handler{
			Contract:privatebridge.Contract{
				Audience:audience,KubernetesIssuer:issuer,
				BFFServiceAccount:subject,AssertionKey:key,
			},
			Tokens:privatebridge.KubernetesTokenReviewer{Client:kube},
			Nonces:privatebridge.KubernetesNonceStore{Client:mgr.GetClient()},
			Identities:privateBFFResolver{controller:&controller.TenantBundleReconciler{
				Client:mgr.GetClient(),APIReader:mgr.GetAPIReader(),Scheme:mgr.GetScheme(),
			}},
		}
		enrollmentHandler:=&privatebridge.EnrollmentHandler{
			Contract:handler.Contract,
			Tokens:handler.Tokens,
			Nonces:handler.Nonces,
			Enrollments:privateBFFEnrollment{controller:&controller.TenantBundleReconciler{
				Client:mgr.GetClient(),APIReader:mgr.GetAPIReader(),Scheme:mgr.GetScheme(),
			}},
		}
		activeHumanHandler:=&privatebridge.ActiveHumanHandler{
			Contract:handler.Contract,
			Tokens:handler.Tokens,
			Nonces:handler.Nonces,
			Accounts:privateBFFActiveHuman{controller:&controller.TenantBundleReconciler{
				Client:mgr.GetClient(),APIReader:mgr.GetAPIReader(),Scheme:mgr.GetScheme(),
			}},
		}
		browserAuthorityHandler:=&privatebridge.BrowserAuthorityHandler{
			Contract:handler.Contract,
			Tokens:handler.Tokens,
			Nonces:handler.Nonces,
			Authorities:privateBrowserAuthorityResolver{controller:&controller.TenantBundleReconciler{
				Client:mgr.GetClient(),APIReader:mgr.GetAPIReader(),Scheme:mgr.GetScheme(),
			}},
		}
		if err:=mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
			listener,err:=net.Listen("tcp",":8090")
			if err!=nil {return err}
			mux:=http.NewServeMux()
			mux.Handle(privatebridge.Path,handler)
			mux.Handle(privatebridge.ActiveHumanPath,activeHumanHandler)
			mux.Handle(privatebridge.EnrollmentPath,enrollmentHandler)
			mux.Handle(privatebridge.BrowserAuthorityPath,browserAuthorityHandler)
			server:=&http.Server{
				Handler:mux,ReadHeaderTimeout:5*time.Second,
				ReadTimeout:8*time.Second,WriteTimeout:8*time.Second,
				IdleTimeout:15*time.Second,MaxHeaderBytes:20*1024,
			}
			go func() {
				<-ctx.Done()
				closeCtx,cancel:=context.WithTimeout(context.Background(),3*time.Second)
				defer cancel()
				_ = server.Shutdown(closeCtx)
			}()
			err=server.Serve(listener)
			if errors.Is(err,http.ErrServerClosed) {return nil}
			return err
		}));err!=nil {
			setupLog.Error(err,"cannot register private BFF identity listener")
			os.Exit(1)
		}
	}
	setupLog.Info("starting TXO Fabric operator")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// privateBFFResolver keeps the security-sensitive operator lookup name
// explicit without widening the privatebridge.Resolver interface.
type privateBFFResolver struct { controller *controller.TenantBundleReconciler }
func (r privateBFFResolver) Resolve(ctx context.Context,tenant,uuid string,now time.Time)(string,error) {
	return r.controller.ResolvePrivateBFFIdentity(ctx,tenant,uuid,now)
}

type privateBrowserAuthorityResolver struct { controller *controller.TenantBundleReconciler }
func (r privateBrowserAuthorityResolver) ResolveBrowserAuthority(ctx context.Context,q privatebridge.BrowserAuthorityRequest,now time.Time)(privatebridge.BrowserAuthorityResponse,error) {
	if r.controller==nil { return privatebridge.BrowserAuthorityResponse{},errors.New("private browser authority unavailable") }
	return r.controller.ResolvePrivateBrowserAuthority(ctx,q,now)
}

type privateBFFActiveHuman struct { controller *controller.TenantBundleReconciler }
func (r privateBFFActiveHuman) VerifyActiveHuman(ctx context.Context,issuer,subject,uuid string,now time.Time) error {
	if r.controller==nil { return errors.New("private active-human verifier unavailable") }
	return r.controller.VerifyPrivateBFFActiveHuman(ctx,issuer,subject,uuid,now)
}

// privateBFFEnrollment keeps enrollment separate from resolution so the latter
// remains read-only and cannot accidentally become an implicit enrollment API.
type privateBFFEnrollment struct { controller *controller.TenantBundleReconciler }
func (r privateBFFEnrollment) Enroll(ctx context.Context,tenant,issuer,subject,uuid string,now time.Time)(string,error) {
	return r.controller.EnrollPrivateBFFIdentity(ctx,tenant,issuer,subject,uuid,now)
}
