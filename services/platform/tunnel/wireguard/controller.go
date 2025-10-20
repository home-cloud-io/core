package wireguard

import (
	"context"
	"fmt"
	"net"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/coreos/go-iptables/iptables"
	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	"github.com/vishvananda/netlink"
)

type WireguardReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	WireguardFinalizer = "wireguard.home-cloud.io/finalizer"
)

func (r *WireguardReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("Reconciling Install")

	// Get the CRD that triggered reconciliation
	obj := &v1.Wireguard{}
	err := r.Get(ctx, req.NamespacedName, obj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			l.Info("Wireguard resource not found. Assuming this means the resource was deleted and so ignoring.")
			return ctrl.Result{}, nil
		}
		l.Info("Failed to get Wireguard resource. Re-running reconcile.")
		return ctrl.Result{}, err
	}

	// if marked for deletion, try to delete/uninstall
	if obj.GetDeletionTimestamp() != nil {
		l.Info("Removing Wireguard interface")
		return ctrl.Result{}, nil
		// TODO: return ctrl.Result{}, r.tryDeletions(ctx, obj)
	}

	l.Info("Reconciling Wireguard")
	return ctrl.Result{}, r.reconcile(ctx, obj)
}

func (r *WireguardReconciler) reconcile(ctx context.Context, wg *v1.Wireguard) error {

	// add link
	inf := &netlink.Wireguard{LinkAttrs: netlink.LinkAttrs{Name: wg.Spec.Name}}
	err := netlink.LinkAdd(inf)
	if err != nil {
		panic(err)
	}
	fmt.Println("link added")

	// add address
	addr, err := netlink.ParseAddr(wg.Spec.Address)
	if err != nil {
		panic(err)
	}
	err = netlink.AddrAdd(inf, addr)
	if err != nil {
		panic(err)
	}
	fmt.Println("address added")

	// parse peers
	peers := make([]wgtypes.PeerConfig, len(wg.Spec.Peers))
	for i, peer := range wg.Spec.Peers {
		publicKey, err := wgtypes.ParseKey(peer.PublicKey)
		if err != nil {
			panic(err)
		}

		// get preshared key if configured
		presharedKey := wgtypes.Key{}
		if peer.PresharedKey != nil {
			presharedKey, err = getKey(ctx, r.Client, *peer.PresharedKey)
			if err != nil {
				panic(err)
			}
		}

		// parse endpoint if configured
		var endpoint *net.UDPAddr
		if peer.Endpoint != nil {
			endpoint, err = net.ResolveUDPAddr("udp", *peer.Endpoint)
			if err != nil {
				panic(err)
			}
		}

		// parse allowed IPs
		allowedIPs := make([]net.IPNet, len(peer.AllowedIPs))
		for i, ip := range peer.AllowedIPs {
			_, ipNet, err := net.ParseCIDR(ip)
			if err != nil {
				panic(err)
			}
			allowedIPs[i] = *ipNet
		}

		// save peer
		peers[i] = wgtypes.PeerConfig{
			PublicKey:                   publicKey,
			PresharedKey:                &presharedKey,
			Endpoint:                    endpoint,
			PersistentKeepaliveInterval: peer.PersistentKeepaliveInterval,
			ReplaceAllowedIPs:           true,
			AllowedIPs:                  allowedIPs,
		}
	}

	// get private key
	privateKey, err := getKey(ctx, r.Client, wg.Spec.PrivateKeySecret)
	if err != nil {
		panic(err)
	}

	// configure device
	wClient, err := wgctrl.New()
	if err != nil {
		panic(err)
	}
	err = wClient.ConfigureDevice(wg.Spec.Name, wgtypes.Config{
		PrivateKey:   &privateKey,
		ListenPort:   ptr.To(wg.Spec.ListenPort),
		ReplacePeers: true,
		Peers:        peers,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("device configured")

	// setup link
	err = netlink.LinkSetUp(inf)
	if err != nil {
		panic(err)
	}
	fmt.Println("link set up")

	// enable nating external traffic
	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		panic(err)
	}
	rule := []string{"-s", wg.Spec.Address, "-o", "eth0", "-j", "MASQUERADE"}
	exists, err := ipt.Exists("nat", "POSTROUTING", rule...)
	if err != nil {
		panic(err)
	}
	if exists {
		return nil
	}
	err = ipt.Append("nat", "POSTROUTING", rule...)
	if err != nil {
		panic(err)
	}
	fmt.Println("iptables configured")

	return nil
}

func getKey(ctx context.Context, kube client.Client, ref v1.SecretReference) (wgtypes.Key, error) {
	secret := &corev1.Secret{}
	err := kube.Get(ctx, types.NamespacedName{
		Name:      ref.Name,
		Namespace: ref.Namespace,
	}, secret)
	if err != nil {
		return wgtypes.Key{}, err
	}
	return wgtypes.ParseKey(string(secret.Data[ref.DataKey]))
}

// SetupWithManager sets up the controller with the Manager.
func (r *WireguardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Wireguard{}).
		Complete(r)
}
