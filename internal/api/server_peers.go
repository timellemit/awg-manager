package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/hoaxisr/awg-manager/internal/managed"
	"github.com/hoaxisr/awg-manager/internal/ndms"
	"github.com/hoaxisr/awg-manager/internal/response"
	"github.com/hoaxisr/awg-manager/internal/storage"
	"github.com/hoaxisr/awg-manager/internal/testing"
)

// ServerAddPeerRequestDTO is the body for POST /servers/{name}/peers.
type ServerAddPeerRequestDTO struct {
	Description string `json:"description" example:"My Phone"`
	TunnelIP    string `json:"tunnelIP" example:"10.0.14.2/32"`
}

// ServerUpdatePeerRequestDTO is the body for PUT /servers/{name}/peers/{pubkey}.
type ServerUpdatePeerRequestDTO struct {
	Description string `json:"description" example:"My Phone"`
	TunnelIP    string `json:"tunnelIP" example:"10.0.14.2/32"`
}

// Subtree dispatches /api/servers/{name}/... operations.
func (h *ServersHandler) Subtree(w http.ResponseWriter, r *http.Request) {
	parts, ok := splitPath(r.URL.EscapedPath(), "/api/servers/")
	if !ok || len(parts) < 2 {
		response.Error(w, "unknown path", "UNKNOWN_PATH")
		return
	}
	name := parts[0]
	if !h.validateName(w, name) {
		return
	}
	switch parts[1] {
	case "nat":
		if len(parts) != 2 {
			response.Error(w, "unknown path", "UNKNOWN_PATH")
			return
		}
		h.SetNAT(w, r, name)
		return
	case "policy":
		if len(parts) != 2 {
			response.Error(w, "unknown path", "UNKNOWN_PATH")
			return
		}
		h.SetPolicy(w, r, name)
		return
	case "peers":
	default:
		response.Error(w, "unknown path", "UNKNOWN_PATH")
		return
	}
	switch len(parts) {
	case 2:
		if r.Method != http.MethodPost {
			response.MethodNotAllowed(w)
			return
		}
		h.AddServerPeer(w, r, name)
	case 3:
		pubkey, err := url.PathUnescape(parts[2])
		if err != nil || !validateWireguardPubkey(pubkey) {
			response.Error(w, "invalid public key", "INVALID_PUBKEY")
			return
		}
		switch r.Method {
		case http.MethodPut:
			h.UpdateServerPeer(w, r, name, pubkey)
		case http.MethodDelete:
			h.DeleteServerPeer(w, r, name, pubkey)
		default:
			response.MethodNotAllowed(w)
		}
	case 4:
		pubkey, err := url.PathUnescape(parts[2])
		if err != nil || !validateWireguardPubkey(pubkey) {
			response.Error(w, "invalid public key", "INVALID_PUBKEY")
			return
		}
		switch parts[3] {
		case "toggle":
			h.ToggleServerPeer(w, r, name, pubkey)
		case "conf":
			h.ServerPeerConf(w, r, name, pubkey)
		default:
			response.Error(w, "unknown path", "UNKNOWN_PATH")
		}
	default:
		response.Error(w, "unknown path", "UNKNOWN_PATH")
	}
}

func validateWireguardPubkey(pubkey string) bool {
	return len(pubkey) == 44 && strings.HasSuffix(pubkey, "=")
}

func (h *ServersHandler) requireListedServer(ctx context.Context, w http.ResponseWriter, name string) (*ndms.WireguardServer, bool) {
	server, err := h.getListedServer(ctx, name)
	if err != nil {
		response.Error(w, err.Error(), "GET_FAILED")
		return nil, false
	}
	if server == nil {
		response.Error(w, "server not found", "NOT_FOUND")
		return nil, false
	}
	return server, true
}

func (h *ServersHandler) requireWGCommands(w http.ResponseWriter) bool {
	if h.commands == nil || h.commands.Wireguard == nil {
		response.Error(w, "ndms commands not initialized", "INTERNAL_ERROR")
		return false
	}
	return true
}

func (h *ServersHandler) AddServerPeer(w http.ResponseWriter, r *http.Request, name string) {
	req, ok := parseJSON[ServerAddPeerRequestDTO](w, r, http.MethodPost)
	if !ok {
		return
	}
	if !h.requireWGCommands(w) {
		return
	}
	server, ok := h.requireListedServer(r.Context(), w, name)
	if !ok {
		return
	}
	if err := h.validateServerPeerTunnelIP(server, req.TunnelIP); err != nil {
		response.Error(w, err.Error(), "INVALID_TUNNEL_IP")
		return
	}
	for _, p := range server.Peers {
		for _, allowed := range p.AllowedIPs {
			if strings.HasPrefix(allowed, strings.TrimSuffix(req.TunnelIP, "/32")) {
				response.Error(w, "tunnel IP already in use", "TUNNEL_IP_IN_USE")
				return
			}
		}
	}

	privKey, pubKey, err := managed.GenerateKeyPair(r.Context())
	if err != nil {
		response.Error(w, err.Error(), "KEYGEN_FAILED")
		return
	}
	psk, err := managed.GeneratePresharedKey(r.Context())
	if err != nil {
		response.Error(w, err.Error(), "KEYGEN_FAILED")
		return
	}
	ip, _, err := net.ParseCIDR(req.TunnelIP)
	if err != nil {
		response.Error(w, "invalid tunnel IP", "INVALID_TUNNEL_IP")
		return
	}

	if err := h.commands.Wireguard.AddPeer(r.Context(), name, pubKey, psk, strings.TrimSpace(req.Description), ip.String(), true); err != nil {
		response.Error(w, err.Error(), "ADD_PEER_FAILED")
		return
	}
	if err := h.settings.SetServerPeerSecret(name, pubKey, storage.ServerPeerSecret{
		PrivateKey:   privKey,
		PresharedKey: psk,
		Description:  req.Description,
		TunnelIP:     req.TunnelIP,
	}); err != nil {
		response.Error(w, err.Error(), "SAVE_FAILED")
		return
	}
	publishInvalidated(h.bus, ResourceServers, "server-peer-added")
	h.writeAll(w, r)
}

func (h *ServersHandler) UpdateServerPeer(w http.ResponseWriter, r *http.Request, name, pubkey string) {
	req, ok := parseJSON[ServerUpdatePeerRequestDTO](w, r, http.MethodPut)
	if !ok {
		return
	}
	if !h.requireWGCommands(w) {
		return
	}
	server, ok := h.requireListedServer(r.Context(), w, name)
	if !ok {
		return
	}
	peer := findServerPeer(server, pubkey)
	if peer == nil {
		response.Error(w, "peer not found", "NOT_FOUND")
		return
	}

	oldIP := peerTunnelHostIP(peer)
	wantIPChange := req.TunnelIP != "" && req.TunnelIP != oldIP+"/32" && req.TunnelIP != oldIP
	if wantIPChange {
		if err := h.validateServerPeerTunnelIP(server, req.TunnelIP); err != nil {
			response.Error(w, err.Error(), "INVALID_TUNNEL_IP")
			return
		}
		newHost, _, _ := net.ParseCIDR(req.TunnelIP)
		if newHost == nil {
			response.Error(w, "invalid tunnel IP", "INVALID_TUNNEL_IP")
			return
		}
		if err := h.commands.Wireguard.UpdatePeerAllowIPs(r.Context(), name, pubkey, oldIP, newHost.String()); err != nil {
			response.Error(w, err.Error(), "UPDATE_PEER_FAILED")
			return
		}
		if sec, ok := h.settings.GetServerPeerSecret(name, pubkey); ok {
			sec.TunnelIP = req.TunnelIP
			_ = h.settings.SetServerPeerSecret(name, pubkey, sec)
		}
	}
	if req.Description != peer.Description {
		if err := h.commands.Wireguard.SetPeerComment(r.Context(), name, pubkey, strings.TrimSpace(req.Description)); err != nil {
			response.Error(w, err.Error(), "UPDATE_PEER_FAILED")
			return
		}
		if sec, ok := h.settings.GetServerPeerSecret(name, pubkey); ok {
			sec.Description = req.Description
			_ = h.settings.SetServerPeerSecret(name, pubkey, sec)
		}
	}
	publishInvalidated(h.bus, ResourceServers, "server-peer-updated")
	h.writeAll(w, r)
}

func (h *ServersHandler) DeleteServerPeer(w http.ResponseWriter, r *http.Request, name, pubkey string) {
	if !h.requireWGCommands(w) {
		return
	}
	server, ok := h.requireListedServer(r.Context(), w, name)
	if !ok {
		return
	}
	if findServerPeer(server, pubkey) == nil {
		response.Error(w, "peer not found", "NOT_FOUND")
		return
	}
	if err := h.commands.Wireguard.RemovePeer(r.Context(), name, pubkey); err != nil {
		response.Error(w, err.Error(), "DELETE_PEER_FAILED")
		return
	}
	_ = h.settings.DeleteServerPeerSecret(name, pubkey)
	publishInvalidated(h.bus, ResourceServers, "server-peer-deleted")
	h.writeAll(w, r)
}

func (h *ServersHandler) ToggleServerPeer(w http.ResponseWriter, r *http.Request, name, pubkey string) {
	req, ok := parseJSON[EnabledToggleRequest](w, r, http.MethodPost)
	if !ok {
		return
	}
	if !h.requireWGCommands(w) {
		return
	}
	server, ok := h.requireListedServer(r.Context(), w, name)
	if !ok {
		return
	}
	if findServerPeer(server, pubkey) == nil {
		response.Error(w, "peer not found", "NOT_FOUND")
		return
	}
	if err := h.commands.Wireguard.SetPeerConnect(r.Context(), name, pubkey, req.Enabled); err != nil {
		response.Error(w, err.Error(), "TOGGLE_FAILED")
		return
	}
	publishInvalidated(h.bus, ResourceServers, "server-peer-toggled")
	h.writeAll(w, r)
}

func (h *ServersHandler) ServerPeerConf(w http.ResponseWriter, r *http.Request, name, pubkey string) {
	if r.Method != http.MethodGet {
		response.MethodNotAllowed(w)
		return
	}
	server, ok := h.requireListedServer(r.Context(), w, name)
	if !ok {
		return
	}
	if findServerPeer(server, pubkey) == nil {
		response.Error(w, "peer not found", "NOT_FOUND")
		return
	}
	sec, ok := h.settings.GetServerPeerSecret(name, pubkey)
	if !ok || sec.PrivateKey == "" {
		response.Error(w, "ключ клиента недоступен (создан вне AWG Manager или через KeenDNS)", "CONF_UNAVAILABLE")
		return
	}
	conf, err := h.generateServerPeerConf(r.Context(), server, pubkey, sec)
	if err != nil {
		response.Error(w, err.Error(), "CONF_FAILED")
		return
	}
	response.Success(w, map[string]string{"conf": conf})
}

func (h *ServersHandler) generateServerPeerConf(ctx context.Context, server *ndms.WireguardServer, pubkey string, sec storage.ServerPeerSecret) (string, error) {
	endpoint, err := h.resolveServerEndpoint(ctx)
	if err != nil {
		return "", err
	}
	tunnelIP := sec.TunnelIP
	if tunnelIP == "" {
		if peer := findServerPeer(server, pubkey); peer != nil {
			host := peerTunnelHostIP(peer)
			if host != "" {
				tunnelIP = host + "/32"
			}
		}
	}
	if tunnelIP == "" {
		return "", fmt.Errorf("peer tunnel IP unknown")
	}
	mtu := server.MTU
	if mtu == 0 {
		mtu = 1420
	}

	var b strings.Builder
	b.WriteString("[Interface]\n")
	b.WriteString(fmt.Sprintf("PrivateKey = %s\n", sec.PrivateKey))
	b.WriteString(fmt.Sprintf("Address = %s\n", tunnelIP))
	b.WriteString("DNS = 1.1.1.1, 8.8.8.8\n")
	b.WriteString(fmt.Sprintf("MTU = %d\n", mtu))

	if h.queries != nil && h.queries.WGServers != nil {
		if ascRaw, err := h.queries.WGServers.GetASCParams(ctx, server.ID, true); err == nil && ascRaw != nil {
			writeServerASCParams(&b, ascRaw)
		}
	}

	b.WriteString("\n[Peer]\n")
	b.WriteString(fmt.Sprintf("PublicKey = %s\n", server.PublicKey))
	if sec.PresharedKey != "" {
		b.WriteString(fmt.Sprintf("PresharedKey = %s\n", sec.PresharedKey))
	}
	b.WriteString(fmt.Sprintf("Endpoint = %s:%d\n", endpoint, server.ListenPort))
	b.WriteString("AllowedIPs = 0.0.0.0/0, ::/0\n")
	b.WriteString("PersistentKeepalive = 25\n")
	return b.String(), nil
}

func (h *ServersHandler) resolveServerEndpoint(ctx context.Context) (string, error) {
	if h.queries != nil && h.queries.KeenDNS != nil {
		if info, err := h.queries.KeenDNS.Get(ctx); err == nil && info != nil && info.Domain != "" {
			return info.Domain, nil
		}
	}
	return testing.GetWANIPWithFallback(ctx, h.queries.WANInterfaceAddress)
}

func writeServerASCParams(b *strings.Builder, raw json.RawMessage) {
	var ext ndms.ASCParamsExtended
	if err := json.Unmarshal(raw, &ext); err != nil || ext.Jc == 0 {
		return
	}
	b.WriteString(fmt.Sprintf("Jc = %d\n", ext.Jc))
	b.WriteString(fmt.Sprintf("Jmin = %d\n", ext.Jmin))
	b.WriteString(fmt.Sprintf("Jmax = %d\n", ext.Jmax))
	b.WriteString(fmt.Sprintf("S1 = %d\n", ext.S1))
	b.WriteString(fmt.Sprintf("S2 = %d\n", ext.S2))
	b.WriteString(fmt.Sprintf("H1 = %s\n", ext.H1))
	b.WriteString(fmt.Sprintf("H2 = %s\n", ext.H2))
	b.WriteString(fmt.Sprintf("H3 = %s\n", ext.H3))
	b.WriteString(fmt.Sprintf("H4 = %s\n", ext.H4))
	if ext.S3 > 0 || ext.S4 > 0 {
		b.WriteString(fmt.Sprintf("S3 = %d\n", ext.S3))
		b.WriteString(fmt.Sprintf("S4 = %d\n", ext.S4))
	}
}

func findServerPeer(server *ndms.WireguardServer, pubkey string) *ndms.WireguardServerPeer {
	for i := range server.Peers {
		if server.Peers[i].PublicKey == pubkey {
			return &server.Peers[i]
		}
	}
	return nil
}

func peerTunnelHostIP(peer *ndms.WireguardServerPeer) string {
	for _, allowed := range peer.AllowedIPs {
		if strings.Contains(allowed, "/32") {
			host, _, err := net.ParseCIDR(allowed)
			if err == nil && host != nil {
				return host.String()
			}
		}
	}
	if len(peer.AllowedIPs) > 0 {
		host, _, err := net.ParseCIDR(peer.AllowedIPs[0])
		if err == nil && host != nil {
			return host.String()
		}
	}
	return ""
}

func (h *ServersHandler) validateServerPeerTunnelIP(server *ndms.WireguardServer, tunnelIP string) error {
	ip, ipNet, err := net.ParseCIDR(tunnelIP)
	if err != nil {
		return fmt.Errorf("invalid tunnel IP (must be CIDR, e.g. 10.0.0.2/32): %w", err)
	}
	serverIP := net.ParseIP(server.Address)
	serverMask := net.IPMask(net.ParseIP(server.Mask).To4())
	if serverIP == nil || serverMask == nil {
		return nil
	}
	serverNet := &net.IPNet{IP: serverIP.Mask(serverMask), Mask: serverMask}
	if !serverNet.Contains(ip) {
		return fmt.Errorf("tunnel IP %s is not in server subnet %s", ip, serverNet)
	}
	if ip.Equal(serverIP) {
		return fmt.Errorf("tunnel IP %s is the server's own address", ip)
	}
	ones, bits := serverNet.Mask.Size()
	if ones < bits-1 {
		if ip.Equal(serverNet.IP) {
			return fmt.Errorf("tunnel IP %s is the network address", ip)
		}
		broadcast := make(net.IP, len(serverNet.IP))
		for i := range serverNet.IP {
			broadcast[i] = serverNet.IP[i] | ^serverNet.Mask[i]
		}
		if ip.Equal(broadcast) {
			return fmt.Errorf("tunnel IP %s is the broadcast address", ip)
		}
	}
	_ = ipNet
	return nil
}

const builtInVPNServerDescription = "Wireguard VPN Server"

func (h *ServersHandler) enrichServerDTO(ctx context.Context, srv ndms.WireguardServer) WireguardServerDTO {
	dto := toWireguardServerDTO(srv)
	dto.BuiltIn = srv.Description == builtInVPNServerDescription
	if _, mode, err := h.readSystemServerNATMode(ctx, srv.ID); err == nil {
		dto.NATEnabled = mode == "full"
		dto.NATMode = mode
	}
	if policy, err := h.readSystemServerPolicy(ctx, srv.ID); err == nil {
		dto.Policy = policy
	}
	if h.queries != nil && h.queries.KeenDNS != nil {
		if info, err := h.queries.KeenDNS.Get(ctx); err == nil && info != nil {
			dto.KeenDNSDomain = info.Domain
		}
	}
	for i := range dto.Peers {
		sec, ok := h.settings.GetServerPeerSecret(srv.ID, dto.Peers[i].PublicKey)
		if ok {
			dto.Peers[i].ConfAvailable = true
			if dto.Peers[i].Description == "" && sec.Description != "" {
				dto.Peers[i].Description = sec.Description
			}
		}
	}
	return dto
}

func toWireguardServerDTO(srv ndms.WireguardServer) WireguardServerDTO {
	peers := make([]WireguardServerPeerDTO, len(srv.Peers))
	for i, p := range srv.Peers {
		peers[i] = WireguardServerPeerDTO{
			PublicKey:     p.PublicKey,
			Description:   p.Description,
			Endpoint:      p.Endpoint,
			AllowedIPs:    p.AllowedIPs,
			RxBytes:       p.RxBytes,
			TxBytes:       p.TxBytes,
			LastHandshake: p.LastHandshake,
			Online:        p.Online,
			Enabled:       p.Enabled,
		}
	}
	return WireguardServerDTO{
		ID:            srv.ID,
		InterfaceName: srv.InterfaceName,
		Description:   srv.Description,
		Status:        srv.Status,
		Connected:     srv.Connected,
		MTU:           srv.MTU,
		Address:       srv.Address,
		Mask:          srv.Mask,
		PublicKey:     srv.PublicKey,
		ListenPort:    srv.ListenPort,
		Peers:         peers,
	}
}
