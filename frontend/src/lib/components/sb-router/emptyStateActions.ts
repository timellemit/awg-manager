import { api } from '$lib/api/client';
import { singboxRouter } from '$lib/stores/singboxRouter';
import type { SingboxRouterDNSServer } from '$lib/types';
import type { CustomMatcherFields } from './addWizardStore';
import type { TemplateGroup } from './templatesData';
import type { SubmitResult } from './templatesActions';
import { submitWizard } from './addWizardActions';
import { mergeAndSaveSettings } from './settingsActions';

export interface FinishSetupArgs {
  tunnelTag: string;
  selectedTemplates: string[];
  customFields: CustomMatcherFields;
  groups: TemplateGroup[];
  tunneledRuleSetTags: string[];
}

export async function finishSetup(args: FinishSetupArgs): Promise<SubmitResult> {
  const result = await submitWizard({
    selectedTemplates: args.selectedTemplates,
    customFields: args.customFields,
    outboundCategory: 'tunnel',
    tunnelTag: args.tunnelTag,
    groups: args.groups,
  });
  try {
    await applyDnsDefaults(args.tunnelTag, args.tunneledRuleSetTags);
  } catch (e) {
    result.failures.push({ id: 'dns', error: e instanceof Error ? e.message : String(e) });
  }
  await api.singboxRouterPutRouteFinal('direct');
  await mergeAndSaveSettings({
    deviceMode: 'all',
    wanAutoDetect: true,
    wanInterface: '',
    snifferEnabled: true,
  });
  await api.singboxRouterEnable();
  await singboxRouter.loadAll();
  return result;
}

const DNS_DIRECT_TAG = 'dns-direct';
const DNS_TUNNEL_TAG = 'dns-tunnel';

/**
 * Запекает leak-safe DNS-дефолты:
 *   - dns-direct (Яндекс 77.88.8.8/udp) → dns.final (нетуннелированный DNS)
 *   - dns-tunnel (Quad9 9.9.9.9/udp, detour=tunnelTag) — для туннелируемых доменов
 *   - DNS-правило rule_set → dns-tunnel
 * Идемпотентно: серверы создаются/обновляются по тегу, правило не дублируется.
 */
export async function applyDnsDefaults(tunnelTag: string, ruleSetTags: string[]): Promise<void> {
  const servers = await api.singboxRouterListDNSServers();
  const tags = new Set(servers.map((s) => s.tag));

  if (!tags.has(DNS_DIRECT_TAG)) {
    await api.singboxRouterAddDNSServer({ tag: DNS_DIRECT_TAG, type: 'udp', server: '77.88.8.8' });
  }

  const tunnelServer: SingboxRouterDNSServer = {
    tag: DNS_TUNNEL_TAG, type: 'udp', server: '9.9.9.9', detour: tunnelTag,
  };
  if (tags.has(DNS_TUNNEL_TAG)) {
    await api.singboxRouterUpdateDNSServer(DNS_TUNNEL_TAG, tunnelServer);
  } else {
    await api.singboxRouterAddDNSServer(tunnelServer);
  }

  if (ruleSetTags.length > 0) {
    const rules = await api.singboxRouterListDNSRules();
    const hasTunnelRule = rules.some((r) => r.server === DNS_TUNNEL_TAG);
    if (!hasTunnelRule) {
      await api.singboxRouterAddDNSRule({ rule_set: ruleSetTags, server: DNS_TUNNEL_TAG });
    }
  }

  const globals = await api.singboxRouterGetDNSGlobals();
  await api.singboxRouterPutDNSGlobals({ final: DNS_DIRECT_TAG, strategy: globals.strategy || 'ipv4_only' });
}
