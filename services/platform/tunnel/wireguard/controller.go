package wireguard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"slices"

	"github.com/coreos/go-iptables/iptables"
	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/home-cloud-io/core/services/platform/operator/api/v1"
	"github.com/home-cloud-io/services/platform/tunnel/stun"
)

type WireguardReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	STUNCtl stun.STUNController

	// state tracks the currently running wireguard interfaces
	state map[types.NamespacedName]*v1.Wireguard
	// locatorState tracks the currently connected locators
	locatorsState map[types.NamespacedName][]*LocatorClient
}

const (
	WireguardFinalizer = "wireguard.home-cloud.io/finalizer"
)

// TODO: do we need a mutex on this whole thing?
func (r *WireguardReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("Reconciling Wireguard interface")

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
		return ctrl.Result{}, r.tryDeletions(ctx, obj)
	}

	current, found := r.state[req.NamespacedName]
	if !found {
		// initialize state for new resource
		current = &v1.Wireguard{}
		r.state[req.NamespacedName] = current
	}

	return ctrl.Result{}, r.reconcile(ctx, current, obj)
}

func (r *WireguardReconciler) reconcile(ctx context.Context, current *v1.Wireguard, new *v1.Wireguard) error {
	log := log.FromContext(ctx)

	// define interface
	inf := &netlink.Wireguard{LinkAttrs: netlink.LinkAttrs{Name: new.Spec.Name}}

	// if the interface name has changed, delete current resources and rebuild
	if new.Spec.Name != current.Spec.Name {
		// remove if this isn't initial setup
		if current.Spec.Name != "" {
			err := r.remove(ctx, current)
			if err != nil {
				return fmt.Errorf("failed to remove old interface: %v", err)
			}
		}

		// add link
		err := netlink.LinkAdd(inf)
		if errors.Is(err, fmt.Errorf("file exists")) {
			return err
		}
		log.Info("link added")
	} else {
		log.Info("link unchanged")
	}

	if new.Spec.Name != current.Spec.Name || new.Spec.Address != current.Spec.Address {
		// add address
		addr, err := netlink.ParseAddr(new.Spec.Address)
		if err != nil {
			return err
		}
		err = netlink.AddrAdd(inf, addr)
		if errors.Is(err, fmt.Errorf("file exists")) {
			return err
		}
		log.Info("address added")
	} else {
		log.Info("address unchanged")
	}

	// NOTE: we always parse peers because they contain keys which come from referenced Secret objects
	// which are not stored in-memory since they can be rotated at any time.
	// TODO: do we need a watcher on the referenced secrets so they can be rotated whenever or should
	// we have an annotation that can be placed on the Wireguard object that triggers a reconcile for keys only?
	peers := make([]wgtypes.PeerConfig, len(new.Spec.Peers))
	for i, peer := range new.Spec.Peers {
		publicKey, err := wgtypes.ParseKey(peer.PublicKey)
		if err != nil {
			return err
		}

		// get preshared key if configured
		presharedKey := wgtypes.Key{}
		if peer.PresharedKey != nil {
			presharedKey, err = GetKey(ctx, r.Client, *peer.PresharedKey, new.Namespace)
			if err != nil {
				return err
			}
		}

		// parse endpoint if configured
		var endpoint *net.UDPAddr
		if peer.Endpoint != nil {
			endpoint, err = net.ResolveUDPAddr("udp", *peer.Endpoint)
			if err != nil {
				return err
			}
		}

		// parse allowed IPs
		allowedIPs := make([]net.IPNet, len(peer.AllowedIPs))
		for i, ip := range peer.AllowedIPs {
			_, ipNet, err := net.ParseCIDR(ip)
			if err != nil {
				return err
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

	// TODO: see note above on peers about why we always get this key
	// get private key
	privateKey, err := GetKey(ctx, r.Client, new.Spec.PrivateKeySecret, new.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get private key: %v", err)
	}

	// configure device
	wClient, err := wgctrl.New()
	if err != nil {
		return err
	}
	err = wClient.ConfigureDevice(new.Spec.Name, wgtypes.Config{
		PrivateKey:   &privateKey,
		ListenPort:   ptr.To(new.Spec.ListenPort),
		ReplacePeers: true,
		Peers:        peers,
	})
	if err != nil {
		return err
	}
	log.Info("device configured")

	if new.Spec.Name != current.Spec.Name {
		// setup link
		err = netlink.LinkSetUp(inf)
		if err != nil {
			return err
		}
		log.Info("link set up")
	} else {
		log.Info("link unchanged")
	}

	if new.Spec.Address != current.Spec.Address || new.Spec.NATInterface != current.Spec.NATInterface {
		// enable nating external traffic
		ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
		if err != nil {
			return err
		}
		rule := []string{"-s", new.Spec.Address, "-o", new.Spec.NATInterface, "-j", "MASQUERADE"}
		exists, err := ipt.Exists("nat", "POSTROUTING", rule...)
		if err != nil {
			return err
		}
		if !exists {
			err = ipt.Append("nat", "POSTROUTING", rule...)
			if err != nil {
				return err
			}
			log.Info("iptables configured")
		}
	} else {
		log.Info("nat unchanged")
	}

	if new.Spec.ListenPort != current.Spec.ListenPort || new.Spec.STUNServer != current.Spec.STUNServer {
		// close existing bind if this isn't initial setup
		if current.Spec.ListenPort != 0 {
			r.STUNCtl.Close(new.Spec.ListenPort)
		}

		stunAddr, err := net.ResolveUDPAddr("udp4", new.Spec.STUNServer)
		if err != nil {
			return err
		}

		// bind to STUN server
		err = r.STUNCtl.Bind(new.Spec.ListenPort, stunAddr)
		if err != nil {
			return err
		}
	} else {
		log.Info("stun unchanged")
	}

	// update locators if the configured locators has changed or if the ID has changed
	if !slices.Equal(new.Spec.Locators, current.Spec.Locators) || new.Spec.ID != current.Spec.ID {
		nn := types.NamespacedName{
			Namespace: new.Namespace,
			Name:      new.Name,
		}

		// cancel previous clients
		if clients, ok := r.locatorsState[nn]; ok {
			for _, client := range clients {
				client.Cancel()
			}
		}

		// connect to new clients
		clients := make([]*LocatorClient, len(new.Spec.Locators))
		for i, address := range new.Spec.Locators {
			run := &LocatorClient{
				Address:            address,
				WireguardReference: nn,
				KubeClient:         r.Client,
			}
			clients[i] = run
			go run.Connect(ctx)
		}

		// save new state
		r.locatorsState[nn] = clients
	} else {
		log.Info("locators unchanged")
	}

	return nil
}

func (r *WireguardReconciler) tryDeletions(ctx context.Context, obj *v1.Wireguard) error {
	if controllerutil.ContainsFinalizer(obj, WireguardFinalizer) {
		err := r.remove(ctx, obj)
		if err != nil {
			return err
		}

		controllerutil.RemoveFinalizer(obj, WireguardFinalizer)
		err = r.Update(ctx, obj)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *WireguardReconciler) remove(ctx context.Context, obj *v1.Wireguard) error {
	log := log.FromContext(ctx)
	inf := &netlink.Wireguard{LinkAttrs: netlink.LinkAttrs{Name: obj.Spec.Name}}

	// cancel locator connections
	if clients, ok := r.locatorsState[types.NamespacedName{
		Namespace: obj.Namespace,
		Name:      obj.Name,
	}]; ok {
		for _, client := range clients {
			client.Cancel()
		}
	}

	// disable nating external traffic
	ipt, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		return err
	}
	rule := []string{"-s", obj.Spec.Address, "-o", "eth0", "-j", "MASQUERADE"}
	exists, err := ipt.Exists("nat", "POSTROUTING", rule...)
	if err != nil {
		return err
	}
	if exists {
		err = ipt.Delete("nat", "POSTROUTING", rule...)
		if err != nil {
			return err
		}
		log.Info("iptables rule removed")
	}

	// stop link
	err = netlink.LinkSetDown(inf)
	if err != nil && err.Error() != "no such device" {
		return err
	}
	log.Info("stopped link")

	// delete address
	addr, err := netlink.ParseAddr(obj.Spec.Address)
	if err != nil {
		return err
	}
	netlink.AddrDel(inf, addr)
	log.Info("deleted address")

	// delete link
	err = netlink.LinkDel(inf)
	if err != nil && err.Error() != "invalid argument" {
		return fmt.Errorf("failed to delete link: %v", err)
	}
	log.Info("deleted link")

	return nil
}

func buildConfig(ctx context.Context, kube client.Client, obj v1.Wireguard) (wgtypes.Config, error) {
	// parse peers
	peers := make([]wgtypes.PeerConfig, len(obj.Spec.Peers))
	for i, peer := range obj.Spec.Peers {
		publicKey, err := wgtypes.ParseKey(peer.PublicKey)
		if err != nil {
			return wgtypes.Config{}, err
		}

		// get preshared key if configured
		presharedKey := wgtypes.Key{}
		if peer.PresharedKey != nil {
			presharedKey, err = GetKey(ctx, kube, *peer.PresharedKey, obj.Namespace)
			if err != nil {
				return wgtypes.Config{}, err
			}
		}

		// parse endpoint if configured
		var endpoint *net.UDPAddr
		if peer.Endpoint != nil {
			endpoint, err = net.ResolveUDPAddr("udp", *peer.Endpoint)
			if err != nil {
				return wgtypes.Config{}, err
			}
		}

		// parse allowed IPs
		allowedIPs := make([]net.IPNet, len(peer.AllowedIPs))
		for i, ip := range peer.AllowedIPs {
			_, ipNet, err := net.ParseCIDR(ip)
			if err != nil {
				return wgtypes.Config{}, err
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
	privateKey, err := GetKey(ctx, kube, obj.Spec.PrivateKeySecret, obj.Namespace)
	if err != nil {
		return wgtypes.Config{}, fmt.Errorf("failed to get private key: %v", err)
	}

	return wgtypes.Config{
		PrivateKey:   &privateKey,
		ListenPort:   ptr.To(obj.Spec.ListenPort),
		ReplacePeers: true,
		Peers:        peers,
	}, nil
}

func GetKey(ctx context.Context, kube client.Client, ref v1.SecretReference, defaultNamespace string) (wgtypes.Key, error) {
	secret := &corev1.Secret{}
	ns := defaultNamespace
	if ref.Namespace != nil {
		ns = *ref.Namespace
	}
	err := kube.Get(ctx, types.NamespacedName{
		Name:      ref.Name,
		Namespace: ns,
	}, secret)
	if err != nil {
		return wgtypes.Key{}, err
	}
	return wgtypes.ParseKey(string(secret.Data[ref.DataKey]))
}

// SetupWithManager sets up the controller with the Manager.
func (r *WireguardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// initialize local state
	r.state = make(map[types.NamespacedName]*v1.Wireguard)
	r.locatorsState = make(map[types.NamespacedName][]*LocatorClient)
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Wireguard{}).
		Complete(r)
}
