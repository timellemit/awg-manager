package singbox

import "context"

// NDMSProxyDependency описывает одно правило NDMS-маршрутизации,
// которое таргетит ProxyN sing-box туннеля. Возвращается API при
// попытке выключить NDMS-Proxy toggle с непустыми зависимостями —
// frontend показывает список и просит пользователя удалить/перенаправить.
type NDMSProxyDependency struct {
	Store          string `json:"store"`          // "clientroute" | "accesspolicy" | "dnsroute" | "hydraroute" | "staticroute"
	RecordID       string `json:"id"`
	Label          string `json:"label"`
	ProxyInterface string `json:"proxyInterface"` // "Proxy0"
	TunnelTag      string `json:"tunnelTag"`      // напр. "us-vless"
}

// DependencyRule — storage-agnostic shape, который каждый scanner
// эмитит для своих записей. Имена сторов помещаются в Scanner.StoreName,
// сами поля правила приводятся к Interface/Label/ID.
type DependencyRule struct {
	ID        string
	Label     string
	Interface string
}

// DependencyScanner — минимальный интерфейс, который каждый
// NDMS-storage обязан реализовать. Конкретные адаптеры живут в своих
// пакетах (clientroute/scanner.go, accesspolicy/scanner.go, …).
type DependencyScanner interface {
	StoreName() string
	ListRulesWithInterface(ctx context.Context) []DependencyRule
}

// FindNDMSProxyDependencies возвращает все правила из scanners,
// чей Interface совпадает с одним из ключей singboxProxies.
// singboxProxies маппит имя ProxyN на tag sing-box туннеля — нужно
// для отображения пользователю ("Proxy0 → us-vless").
//
// Возвращает nil, если зависимостей нет (handler смотрит len()).
func FindNDMSProxyDependencies(ctx context.Context, singboxProxies map[string]string, scanners []DependencyScanner) []NDMSProxyDependency {
	if len(singboxProxies) == 0 {
		return nil
	}
	var out []NDMSProxyDependency
	for _, sc := range scanners {
		for _, rule := range sc.ListRulesWithInterface(ctx) {
			tag, hit := singboxProxies[rule.Interface]
			if !hit {
				continue
			}
			out = append(out, NDMSProxyDependency{
				Store:          sc.StoreName(),
				RecordID:       rule.ID,
				Label:          rule.Label,
				ProxyInterface: rule.Interface,
				TunnelTag:      tag,
			})
		}
	}
	return out
}
