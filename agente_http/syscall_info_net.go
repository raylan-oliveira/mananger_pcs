package main

import (
	"net"
)

// Estruturas para informações de rede via syscall
type ipAdapterInfo struct {
	Next                *ipAdapterInfo
	ComboIndex          uint32
	AdapterName         [260]byte
	Description         [132]byte
	AddressLength       uint32
	Address             [8]byte
	Index               uint32
	Type                uint32
	DhcpEnabled         uint32
	CurrentIpAddress    *ipAddrString
	IpAddressList       ipAddrString
	GatewayList         ipAddrString
	DhcpServer          ipAddrString
	HaveWins            bool
	PrimaryWinsServer   ipAddrString
	SecondaryWinsServer ipAddrString
	LeaseObtained       int64
	LeaseExpires        int64
}

type ipAddrString struct {
	Next      *ipAddrString
	IpAddress [16]byte
	IpMask    [16]byte
	Context   uint32
}

func getNetworkInfoSyscall() map[string]interface{} {
	info := make(map[string]interface{})

	// Inicializar as DLLs e procedimentos
	err := initWindowsDLLs()
	if err != nil {
		info["erro"] = err.Error()
		return info
	}

	// Verificar se temos os procedimentos necessários para rede
	if iphlpapiDLL == nil || getNetworkParamsFn == nil {
		info["aviso"] = "Funções de rede não disponíveis completamente"
	}

	// Usar a biblioteca net padrão do Go para obter interfaces
	// Isso é mais confiável do que tentar usar syscalls diretas para redes
	interfaces, err := net.Interfaces()
	if err == nil {
		var networkInterfaces []map[string]interface{}

		for _, iface := range interfaces {
			// Ignorar interfaces de loopback
			if iface.Flags&net.FlagLoopback != 0 {
				continue
			}

			netInterface := make(map[string]interface{})
			netInterface["nome"] = iface.Name
			netInterface["mac"] = iface.HardwareAddr.String()

			// Obter endereços IP
			addrs, err := iface.Addrs()
			if err == nil {
				var ipv4 []string
				var ipv6 []string

				for _, addr := range addrs {
					if ipnet, ok := addr.(*net.IPNet); ok {
						if ip4 := ipnet.IP.To4(); ip4 != nil {
							ipv4 = append(ipv4, ip4.String())
						} else {
							ipv6 = append(ipv6, ipnet.IP.String())
						}
					}
				}

				netInterface["ipv4"] = ipv4
				netInterface["ipv6"] = ipv6
			}

			// Só adicionar interfaces que têm pelo menos um endereço IP
			if len(netInterface["ipv4"].([]string)) > 0 || len(netInterface["ipv6"].([]string)) > 0 {
				networkInterfaces = append(networkInterfaces, netInterface)
			}
		}

		info["interfaces"] = networkInterfaces
	}

	// Tentar obter informações de DNS via syscall
	if iphlpapiDLL != nil && getNetworkParamsFn != nil {
		// Implementação simplificada - na prática, obter servidores DNS
		// via syscall é complexo e requer estruturas adicionais
		info["dns_obtido_via"] = "syscall_parcial"
	}

	info["metodo"] = "net_go_com_syscall_parcial"
	return info
}
