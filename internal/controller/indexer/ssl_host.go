package indexer

import (
	"context"
	"sort"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	sslutil "github.com/apache/apisix-ingress-controller/internal/ssl"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
)

var tlsHostIndexLogger = ctrl.Log.WithName("tls-host-indexer")

// TLSSecretAccessor retrieves a Kubernetes Secret for TLS host indexing.
type TLSSecretAccessor func(context.Context, types.NamespacedName) (*corev1.Secret, error)

// TLSHostIndexers bundles the indexer functions backed by a shared Secret accessor.
type TLSHostIndexers struct {
	getSecret TLSSecretAccessor
}

// NewTLSHostIndexers builds index functions that share the provided Secret accessor.
func NewTLSHostIndexers(accessor TLSSecretAccessor) *TLSHostIndexers {
	return &TLSHostIndexers{
		getSecret: accessor,
	}
}

func setupTLSHostIndexer(mgr ctrl.Manager) error {
	indexers := NewTLSHostIndexers(newClientTLSSecretAccessor(mgr.GetClient()))
	fieldIndexer := mgr.GetFieldIndexer()
	for _, registration := range []struct {
		Obj  client.Object
		Func client.IndexerFunc
	}{
		{Obj: &gatewayv1.Gateway{}, Func: indexers.GatewayTLSHostIndexFunc},
		{Obj: &networkingv1.Ingress{}, Func: indexers.IngressTLSHostIndexFunc},
		{Obj: &apiv2.ApisixTls{}, Func: indexers.ApisixTlsHostIndexFunc},
	} {
		if err := fieldIndexer.IndexField(context.Background(), registration.Obj, TLSHostIndexRef, registration.Func); err != nil {
			return err
		}
	}
	return nil
}

// newClientTLSSecretAccessor returns a TLSSecretAccessor backed by a controller-runtime client.
func newClientTLSSecretAccessor(c client.Reader) TLSSecretAccessor {
	return func(ctx context.Context, nn types.NamespacedName) (*corev1.Secret, error) {
		var secret corev1.Secret
		if err := c.Get(ctx, nn, &secret); err != nil {
			return nil, err
		}
		return &secret, nil
	}
}

// GatewayTLSHostIndexFunc indexes Gateways by their TLS SNI hosts.
func (i *TLSHostIndexers) GatewayTLSHostIndexFunc(rawObj client.Object) []string {
	gateway, ok := rawObj.(*gatewayv1.Gateway)
	if !ok {
		return nil
	}
	if len(gateway.Spec.Listeners) == 0 {
		return nil
	}

	hosts := make(map[string]struct{})

	for _, listener := range gateway.Spec.Listeners {
		if listener.TLS == nil || len(listener.TLS.CertificateRefs) == 0 {
			continue
		}

		candidates := make([]string, 0)
		if listener.Hostname != nil && *listener.Hostname != "" {
			candidates = append(candidates, string(*listener.Hostname))
			candidates = sslutil.NormalizeHosts(candidates)
		}

		if len(candidates) == 0 {
			candidates = append(candidates, i.collectHostsFromSecretRefs(gateway.Namespace, listener.TLS.CertificateRefs)...)
		}

		for _, host := range candidates {
			if host == "" {
				continue
			}
			hosts[host] = struct{}{}
		}
	}

	return hostSetToSlice(hosts)
}

// IngressTLSHostIndexFunc indexes Ingresses by their TLS SNI hosts.
func (i *TLSHostIndexers) IngressTLSHostIndexFunc(rawObj client.Object) []string {
	ingress, ok := rawObj.(*networkingv1.Ingress)
	if !ok {
		return nil
	}
	if len(ingress.Spec.TLS) == 0 {
		return nil
	}

	hosts := make(map[string]struct{})
	for _, tls := range ingress.Spec.TLS {
		if tls.SecretName == "" {
			continue
		}

		candidates := sslutil.NormalizeHosts(tls.Hosts)
		if len(candidates) == 0 {
			nn := types.NamespacedName{Namespace: ingress.Namespace, Name: tls.SecretName}
			candidates = i.collectHostsFromSecret(nn)
		}
		for _, host := range candidates {
			if host == "" {
				continue
			}
			hosts[host] = struct{}{}
		}
	}

	return hostSetToSlice(hosts)
}

// ApisixTlsHostIndexFunc indexes ApisixTls resources by their declared TLS hosts.
func (i *TLSHostIndexers) ApisixTlsHostIndexFunc(rawObj client.Object) []string {
	tls, ok := rawObj.(*apiv2.ApisixTls)
	if !ok {
		return nil
	}
	if len(tls.Spec.Hosts) == 0 {
		return nil
	}

	hostSet := make(map[string]struct{}, len(tls.Spec.Hosts))
	for _, host := range tls.Spec.Hosts {
		for _, normalized := range sslutil.NormalizeHosts([]string{string(host)}) {
			if normalized == "" {
				continue
			}
			hostSet[normalized] = struct{}{}
		}
	}
	return hostSetToSlice(hostSet)
}

func (i *TLSHostIndexers) collectHostsFromSecretRefs(defaultNamespace string, refs []gatewayv1.SecretObjectReference) []string {
	hostSet := make(map[string]struct{})
	for _, ref := range refs {
		if ref.Kind != nil && *ref.Kind != internaltypes.KindSecret {
			continue
		}
		if ref.Group != nil && string(*ref.Group) != corev1.GroupName {
			continue
		}
		secretNN := types.NamespacedName{
			Namespace: defaultNamespace,
			Name:      string(ref.Name),
		}
		if ref.Namespace != nil && *ref.Namespace != "" {
			secretNN.Namespace = string(*ref.Namespace)
		}

		for _, host := range i.collectHostsFromSecret(secretNN) {
			hostSet[host] = struct{}{}
		}
	}

	return hostSetToSlice(hostSet)
}

func (i *TLSHostIndexers) collectHostsFromSecret(nn types.NamespacedName) []string {
	if i == nil || i.getSecret == nil {
		return nil
	}
	secret, err := i.getSecret(context.Background(), nn)
	if err != nil {
		tlsHostIndexLogger.Error(err, "failed to read secret while building TLS host index", "secret", nn)
		return nil
	}

	cert, err := sslutil.ExtractCertificate(secret)
	if err != nil {
		tlsHostIndexLogger.Error(err, "failed to extract certificate while building TLS host index", "secret", nn)
		return nil
	}

	hosts, err := sslutil.ExtractHostsFromCertificate(cert)
	if err != nil {
		tlsHostIndexLogger.Error(err, "failed to extract hosts from certificate while building TLS host index", "secret", nn)
		return nil
	}

	return sslutil.NormalizeHosts(hosts)
}

func hostSetToSlice(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	result := make([]string, 0, len(set))
	for host := range set {
		result = append(result, host)
	}
	sort.Strings(result)
	return result
}
