package controller

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	fabricv1alpha1 "github.com/charchess/vixens/apps/60-services/txo-fabric/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	authentikNamespace               = "auth"
	authentikBlueprintConfigMapName  = "txo-fabric-authentik-blueprints"
	authentikBlueprintKey            = "txo-fabric-tenants.yaml"
)

var iamGroupKeyPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
var domainSuffixPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9.-]{1,251}[a-z0-9])?$`)

func validateTenantIAM(bundle *fabricv1alpha1.TenantBundle) error {
	if bundle.Spec.HumanAccess == nil || bundle.Spec.HumanAccess.Web == nil {
		return nil
	}
	web := bundle.Spec.HumanAccess.Web
	if len(web.IAMGroups) == 0 {
		return fmt.Errorf("TenantBundle %q humanAccess.web requires at least one iamGroups entry", bundle.Name)
	}
	seen := map[string]struct{}{}
	for _, group := range web.IAMGroups {
		group = strings.TrimSpace(group)
		if !iamGroupKeyPattern.MatchString(group) {
			return fmt.Errorf("TenantBundle %q humanAccess.web iamGroup %q must be a lowercase DNS-like key", bundle.Name, group)
		}
		if _, ok := seen[group]; ok {
			return fmt.Errorf("TenantBundle %q humanAccess.web repeats iamGroup %q", bundle.Name, group)
		}
		seen[group] = struct{}{}
	}
	domain := strings.Trim(strings.ToLower(strings.TrimSpace(web.DomainSuffix)), ".")
	if !domainSuffixPattern.MatchString(domain) || strings.Contains(domain, "..") {
		return fmt.Errorf("TenantBundle %q humanAccess.web domainSuffix %q is not a valid DNS suffix", bundle.Name, web.DomainSuffix)
	}
	issuer, err := url.Parse(strings.TrimSpace(web.OIDC.Issuer))
	if err != nil || issuer.Scheme != "https" || issuer.Hostname() == "" {
		return fmt.Errorf("TenantBundle %q humanAccess OIDC issuer must be an absolute https URL", bundle.Name)
	}
	app := authentikApplicationName(bundle.Name)
	if issuer.Path != "/application/o/"+app+"/" {
		return fmt.Errorf("TenantBundle %q OIDC issuer path %q must match generated Authentik application %q", bundle.Name, issuer.Path, app)
	}
	return nil
}

func authentikApplicationName(tenantName string) string {
	return "txo-fabric-" + tenantName
}

func authentikGroupName(tenantName, group string) string {
	return authentikApplicationName(tenantName) + "-" + group
}

func activeIAMBundles(bundles []fabricv1alpha1.TenantBundle) []fabricv1alpha1.TenantBundle {
	active := make([]fabricv1alpha1.TenantBundle, 0, len(bundles))
	for i := range bundles {
		bundle := bundles[i]
		if !bundle.DeletionTimestamp.IsZero() || bundle.Spec.HumanAccess == nil || bundle.Spec.HumanAccess.Web == nil {
			continue
		}
		if validateTenantIAM(&bundle) != nil {
			continue
		}
		active = append(active, bundle)
	}
	sort.Slice(active, func(i, j int) bool { return active[i].Name < active[j].Name })
	return active
}

func renderAuthentikBlueprint(bundles []fabricv1alpha1.TenantBundle) string {
	active := activeIAMBundles(bundles)
	if len(active) == 0 {
		return "version: 1\nmetadata:\n  name: txo-fabric-tenants-generated\nentries: []\n"
	}
	var out strings.Builder
	out.WriteString("version: 1\nmetadata:\n  name: txo-fabric-tenants-generated\nentries:\n")
	// One shared source mapping is linked to every tenant provider. The claim
	// comes from Authentik's immutable user UUID, never a mutable username or sub.
	// The Fabric BFF must explicitly request txo_fabric_identity on login.
	out.WriteString("  - model: authentik_providers_oauth2.scopemapping\n    id: txo-fabric-human-uuid-scope\n    identifiers:\n      name: TXO Fabric verified human identity\n    attrs:\n      scope_name: txo_fabric_identity\n      description: Fabric signed immutable Authentik user UUID\n      expression: |\n        return {\"txo_fabric_user_uuid\": str(request.user.uuid)}\n\n")

	proxyProviderIDs := make([]string, 0)
	for i := range active {
		bundle := &active[i]
		web := bundle.Spec.HumanAccess.Web
		app := authentikApplicationName(bundle.Name)
		groups := append([]string(nil), web.IAMGroups...)
		sort.Strings(groups)

		for _, group := range groups {
			groupName := authentikGroupName(bundle.Name, group)
			fmt.Fprintf(&out, "  - model: authentik_core.group\n    id: %s-group\n    identifiers:\n      name: %s\n\n", groupName, groupName)
		}

		fmt.Fprintf(&out, "  - model: authentik_providers_oauth2.oauth2provider\n    id: %s-provider\n    identifiers:\n      name: %s\n    attrs:\n      client_id: %s\n      client_type: public\n      include_claims_in_id_token: true\n      authentication_flow: !Find [authentik_flows.flow, [slug, default-authentication-flow]]\n      authorization_flow: !Find [authentik_flows.flow, [slug, default-provider-authorization-explicit-consent]]\n      invalidation_flow: !Find [authentik_flows.flow, [slug, default-provider-invalidation-flow]]\n      signing_key: !Find [authentik_crypto.certificatekeypair, [name, authentik Self-signed Certificate]]\n      grant_types:\n        - authorization_code\n        - refresh_token\n      property_mappings:\n        - !Find [authentik_providers_oauth2.scopemapping, [managed, goauthentik.io/providers/oauth2/scope-openid]]\n        - !Find [authentik_providers_oauth2.scopemapping, [managed, goauthentik.io/providers/oauth2/scope-email]]\n        - !Find [authentik_providers_oauth2.scopemapping, [managed, goauthentik.io/providers/oauth2/scope-profile]]\n        - !KeyOf txo-fabric-human-uuid-scope\n      redirect_uris:\n        - matching_mode: regex\n          url: '^https://[a-z0-9-]+-%s\\.%s/auth/callback$'\n          redirect_uri_type: authorization\n\n", app, app, strconv.Quote(web.OIDC.ClientID), regexp.QuoteMeta(bundle.Name), regexp.QuoteMeta(strings.Trim(strings.ToLower(strings.TrimSpace(web.DomainSuffix)), ".")))

		fmt.Fprintf(&out, "  - model: authentik_core.application\n    id: %s-app\n    identifiers:\n      slug: %s\n    attrs:\n      name: %s\n      policy_engine_mode: any\n      provider: !KeyOf %s-provider\n\n", app, app, strconv.Quote("TXO Fabric "+bundle.Spec.DisplayName), app)

		for index, group := range groups {
			groupName := authentikGroupName(bundle.Name, group)
			fmt.Fprintf(&out, "  - model: authentik_policies.policybinding\n    identifiers:\n      target: !KeyOf %s-app\n      order: %d\n    attrs:\n      group: !KeyOf %s-group\n\n", app, index*10, groupName)
		}

		if bundle.Spec.Memory.Hindsight != nil && bundle.Spec.Memory.Hindsight.HumanAccess {
			proxyID := app + "-hindsight-provider"
			proxyProviderIDs = append(proxyProviderIDs, proxyID)
			domain := strings.Trim(strings.ToLower(strings.TrimSpace(web.DomainSuffix)), ".")
			fmt.Fprintf(&out, "  - model: authentik_providers_proxy.proxyprovider\n    id: %s\n    identifiers:\n      name: %s-hindsight\n    attrs:\n      mode: forward_single\n      external_host: https://hindsight-%s.%s\n      authentication_flow: !Find [authentik_flows.flow, [slug, default-authentication-flow]]\n      authorization_flow: !Find [authentik_flows.flow, [slug, default-provider-authorization-implicit-consent]]\n      invalidation_flow: !Find [authentik_flows.flow, [slug, default-provider-invalidation-flow]]\n\n", proxyID, app, bundle.Name, domain)
			fmt.Fprintf(&out, "  - model: authentik_core.application\n    id: %s-hindsight-app\n    identifiers:\n      slug: %s-hindsight\n    attrs:\n      name: %s\n      policy_engine_mode: any\n      provider: !KeyOf %s\n\n", app, app, strconv.Quote("TXO Fabric "+bundle.Spec.DisplayName+" Hindsight"), proxyID)
			for index, group := range groups {
				groupName := authentikGroupName(bundle.Name, group)
				fmt.Fprintf(&out, "  - model: authentik_policies.policybinding\n    identifiers:\n      target: !KeyOf %s-hindsight-app\n      order: %d\n    attrs:\n      group: !KeyOf %s-group\n\n", app, index*10, groupName)
			}
		}
	}

	if len(proxyProviderIDs) > 0 {
		sort.Strings(proxyProviderIDs)
		out.WriteString("  - model: authentik_outposts.outpost\n    identifiers:\n      managed: goauthentik.io/outposts/embedded\n    attrs:\n      name: authentik Embedded Outpost\n      type: proxy\n      providers:\n")
		for _, providerID := range proxyProviderIDs {
			fmt.Fprintf(&out, "        - !KeyOf %s\n", providerID)
		}
	}
	return out.String()
}

func (r *TenantBundleReconciler) reconcileAuthentikBlueprint(ctx context.Context) error {
	var bundles fabricv1alpha1.TenantBundleList
	if err := r.List(ctx, &bundles); err != nil {
		return err
	}
	active := activeIAMBundles(bundles.Items)
	configMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: authentikBlueprintConfigMapName, Namespace: authentikNamespace}}
	if len(active) == 0 {
		if err := r.Get(ctx, client.ObjectKeyFromObject(configMap), configMap); err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}
	}
	content := renderAuthentikBlueprint(active)
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		configMap.Labels = map[string]string{
			LabelPartOf:                    "txo-fabric",
			LabelManaged:                   "true",
			"app.kubernetes.io/name":      "txo-fabric-authentik-blueprints",
			"app.kubernetes.io/component": "tenant-iam",
		}
		configMap.Data = map[string]string{authentikBlueprintKey: content}
		return nil
	})
	return err
}

func (r *TenantBundleReconciler) tenantBundleRequestsForAuthentikBlueprint(ctx context.Context, obj client.Object) []reconcile.Request {
	if obj.GetNamespace() != authentikNamespace || obj.GetName() != authentikBlueprintConfigMapName {
		return nil
	}
	var bundles fabricv1alpha1.TenantBundleList
	if err := r.List(ctx, &bundles); err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0, len(bundles.Items))
	for i := range bundles.Items {
		if bundles.Items[i].Spec.HumanAccess == nil || bundles.Items[i].Spec.HumanAccess.Web == nil {
			continue
		}
		requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKey{Name: bundles.Items[i].Name}})
	}
	return requests
}
