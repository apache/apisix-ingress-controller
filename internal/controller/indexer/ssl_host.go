package indexer

import (
	"context"
	"fmt"
	"sort"
	"sync"

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

var (
	tlsHostIndexLogger   = ctrl.Log.WithName("tls-host-indexer")
	tlsSecretAccessor    TLSSecretAccessor
	tlsSecretAccessorMux sync.RWMutex
)

// TLSSecretAccessor abstracts the retrieval of Kubernetes Secrets so the TLS host
// indexers can be reused in unit tests with an in-memory secret store.
type TLSSecretAccessor interface {
	Get(ctx context.Context, nn types.NamespacedName) (*corev1.Secret, error)
}

// TLSSecretAccessorFunc wraps a function as a TLSSecretAccessor.
type TLSSecretAccessorFunc func(context.Context, types.NamespacedName) (*corev1.Secret, error)

// Get implements TLSSecretAccessor.
func (f TLSSecretAccessorFunc) Get(ctx context.Context, nn types.NamespacedName) (*corev1.Secret, error) {
	return f(ctx, nn)
}

// SetTLSHostSecretAccessor configures the secret accessor used by TLS host indexers.
func SetTLSHostSecretAccessor(accessor TLSSecretAccessor) {
	tlsSecretAccessorMux.Lock()
	defer tlsSecretAccessorMux.Unlock()
	tlsSecretAccessor = accessor
}

// ResetTLSHostSecretAccessor clears the secret accessor. Primarily used in tests.
func ResetTLSHostSecretAccessor() {
	SetTLSHostSecretAccessor(nil)
}

func getTLSHostSecretAccessor() TLSSecretAccessor {
	tlsSecretAccessorMux.RLock()
	defer tlsSecretAccessorMux.RUnlock()
	return tlsSecretAccessor
}

func setupTLSHostIndexer(mgr ctrl.Manager) error {
	SetTLSHostSecretAccessor(newClientTLSSecretAccessor(mgr.GetClient()))

	fieldIndexer := mgr.GetFieldIndexer()
	for _, registration := range []struct {
		Obj  client.Object
		Func client.IndexerFunc
	}{
		{Obj: &gatewayv1.Gateway{}, Func: GatewayTLSHostIndexFunc},
		{Obj: &networkingv1.Ingress{}, Func: IngressTLSHostIndexFunc},
		{Obj: &apiv2.ApisixTls{}, Func: ApisixTlsHostIndexFunc},
	} {
		if err := fieldIndexer.IndexField(context.Background(), registration.Obj, TLSHostIndexRef, registration.Func); err != nil {
			return err
		}
	}
	return nil
}

// newClientTLSSecretAccessor returns a TLSSecretAccessor backed by a controller-runtime client.
func newClientTLSSecretAccessor(c client.Reader) TLSSecretAccessor {
	return TLSSecretAccessorFunc(func(ctx context.Context, nn types.NamespacedName) (*corev1.Secret, error) {
		var secret corev1.Secret
		if err := c.Get(ctx, nn, &secret); err != nil {
			return nil, err
		}
		return &secret, nil
	})
}

// GatewayTLSHostIndexFunc indexes Gateways by their TLS SNI hosts.
func GatewayTLSHostIndexFunc(rawObj client.Object) []string {
	gateway, ok := rawObj.(*gatewayv1.Gateway)
	if !ok {
		return nil
	}
	if len(gateway.Spec.Listeners) == 0 {
		return nil
	}

	hosts := make(map[string]struct{})
	accessor := getTLSHostSecretAccessor()

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
			candidates = append(candidates, collectHostsFromSecretRefs(accessor, gateway.Namespace, listener.TLS.CertificateRefs)...)
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
func IngressTLSHostIndexFunc(rawObj client.Object) []string {
	ingress, ok := rawObj.(*networkingv1.Ingress)
	if !ok {
		return nil
	}
	if len(ingress.Spec.TLS) == 0 {
		return nil
	}

	hosts := make(map[string]struct{})
	accessor := getTLSHostSecretAccessor()
	for _, tls := range ingress.Spec.TLS {
		if tls.SecretName == "" {
			continue
		}

		candidates := sslutil.NormalizeHosts(tls.Hosts)
		if len(candidates) == 0 {
			nn := types.NamespacedName{Namespace: ingress.Namespace, Name: tls.SecretName}
			candidates = collectHostsFromSecret(accessor, nn)
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
func ApisixTlsHostIndexFunc(rawObj client.Object) []string {
	tls, ok := rawObj.(*apiv2.ApisixTls)
	if !ok {
		return nil
	}
	if len(tls.Spec.Hosts) == 0 {
		return nil
	}

	hostValues := make([]string, 0, len(tls.Spec.Hosts))
	for _, host := range tls.Spec.Hosts {
		hostValues = append(hostValues, string(host))
	}
	return hostSetToSlice(sliceToHostSet(sslutil.NormalizeHosts(hostValues)))
}

func collectHostsFromSecretRefs(accessor TLSSecretAccessor, defaultNamespace string, refs []gatewayv1.SecretObjectReference) []string {
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

		for _, host := range collectHostsFromSecret(accessor, secretNN) {
			hostSet[host] = struct{}{}
		}
	}

	return hostSetToSlice(hostSet)
}

func collectHostsFromSecret(accessor TLSSecretAccessor, nn types.NamespacedName) []string {
	if accessor == nil {
		return nil
	}
	secret, err := accessor.Get(context.Background(), nn)
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

func sliceToHostSet(hosts []string) map[string]struct{} {
	if len(hosts) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(hosts))
	for _, host := range hosts {
		if host == "" {
			continue
		}
		set[host] = struct{}{}
	}
	return set
}

// NewStaticTLSSecretAccessor builds an accessor backed by an in-memory set of Secrets.
func NewStaticTLSSecretAccessor(secrets []*corev1.Secret) TLSSecretAccessor {
	secretStore := make(map[types.NamespacedName]*corev1.Secret, len(secrets))
	for _, secret := range secrets {
		if secret == nil {
			continue
		}
		key := types.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}
		secretCopy := secret.DeepCopy()
		secretStore[key] = secretCopy
	}
	return TLSSecretAccessorFunc(func(ctx context.Context, nn types.NamespacedName) (*corev1.Secret, error) {
		if secret, ok := secretStore[nn]; ok {
			return secret.DeepCopy(), nil
		}
		return nil, fmt.Errorf("secret %s not found", nn.String())
	})
}
