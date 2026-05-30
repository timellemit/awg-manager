import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('$lib/api/client', () => ({
  api: {
    singboxRouterEnable: vi.fn(),
    singboxRouterPutRouteFinal: vi.fn(),
    singboxRouterListDNSServers: vi.fn(async () => []),
    singboxRouterAddDNSServer: vi.fn(),
    singboxRouterUpdateDNSServer: vi.fn(),
    singboxRouterListDNSRules: vi.fn(async () => []),
    singboxRouterAddDNSRule: vi.fn(),
    singboxRouterGetDNSGlobals: vi.fn(async () => ({ final: '', strategy: 'ipv4_only' })),
    singboxRouterPutDNSGlobals: vi.fn(),
  },
}));

vi.mock('./addWizardActions', () => ({
  submitWizard: vi.fn(async () => ({ successes: ['svc:netflix'], failures: [] })),
}));

vi.mock('./settingsActions', () => ({
  mergeAndSaveSettings: vi.fn(async () => {}),
}));

vi.mock('$lib/stores/singboxRouter', () => {
  const settings = { subscribe: vi.fn(() => () => {}) };
  return {
    singboxRouter: {
      settings,
      loadAll: vi.fn(async () => {}),
    },
  };
});

import { api } from '$lib/api/client';
import { singboxRouter } from '$lib/stores/singboxRouter';
import { submitWizard } from './addWizardActions';
import { mergeAndSaveSettings } from './settingsActions';
import { finishSetup, applyDnsDefaults } from './emptyStateActions';

describe('emptyStateActions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('finishSetup: правила(tunnel) → final=direct → bake defaults(all) → enable → loadAll', async () => {
    const empty = {
      domainSuffix: '', ipCidr: '', sourceIpCidr: '', port: '', ruleSetTags: new Set<string>(),
    };
    const res = await finishSetup({
      tunnelTag: 'wg-nl',
      selectedTemplates: ['svc:netflix'],
      customFields: empty,
      groups: [],
    });
    expect(submitWizard).toHaveBeenCalledWith(
      expect.objectContaining({ outboundCategory: 'tunnel', tunnelTag: 'wg-nl', selectedTemplates: ['svc:netflix'] }),
    );
    expect(api.singboxRouterPutRouteFinal).toHaveBeenCalledWith('direct');
    expect(mergeAndSaveSettings).toHaveBeenCalledWith(
      expect.objectContaining({ deviceMode: 'all', wanAutoDetect: true, wanInterface: '', snifferEnabled: true }),
    );
    expect(api.singboxRouterEnable).toHaveBeenCalled();
    expect(singboxRouter.loadAll).toHaveBeenCalled();
    expect(res.successes).toContain('svc:netflix');
  });
});

describe('applyDnsDefaults', () => {
  beforeEach(() => vi.clearAllMocks());

  it('создаёт dns-direct (Яндекс), dns-tunnel (Quad9, detour), DNS-правило, ставит final=dns-direct', async () => {
    (api.singboxRouterListDNSServers as ReturnType<typeof vi.fn>).mockResolvedValue([]);
    (api.singboxRouterListDNSRules as ReturnType<typeof vi.fn>).mockResolvedValue([]);
    (api.singboxRouterGetDNSGlobals as ReturnType<typeof vi.fn>).mockResolvedValue({ final: '', strategy: 'ipv4_only' });

    await applyDnsDefaults('my-selector', ['geosite-netflix', 'geosite-youtube']);

    expect(api.singboxRouterAddDNSServer).toHaveBeenCalledWith(
      expect.objectContaining({ tag: 'dns-direct', type: 'udp', server: '77.88.8.8' }),
    );
    expect(api.singboxRouterAddDNSServer).toHaveBeenCalledWith(
      expect.objectContaining({ tag: 'dns-tunnel', type: 'udp', server: '9.9.9.9', detour: 'my-selector' }),
    );
    expect(api.singboxRouterAddDNSRule).toHaveBeenCalledWith(
      expect.objectContaining({ rule_set: ['geosite-netflix', 'geosite-youtube'], server: 'dns-tunnel' }),
    );
    expect(api.singboxRouterPutDNSGlobals).toHaveBeenCalledWith(
      expect.objectContaining({ final: 'dns-direct', strategy: 'ipv4_only' }),
    );
  });

  it('идемпотентно: существующий dns-tunnel обновляется, dns-direct не дублируется, правило не повторяется', async () => {
    (api.singboxRouterListDNSServers as ReturnType<typeof vi.fn>).mockResolvedValue([
      { tag: 'dns-direct', type: 'udp', server: '77.88.8.8' },
      { tag: 'dns-tunnel', type: 'udp', server: '9.9.9.9', detour: 'old-tunnel' },
    ]);
    (api.singboxRouterListDNSRules as ReturnType<typeof vi.fn>).mockResolvedValue([
      { rule_set: ['geosite-netflix'], server: 'dns-tunnel' },
    ]);
    (api.singboxRouterGetDNSGlobals as ReturnType<typeof vi.fn>).mockResolvedValue({ final: '', strategy: '' });

    await applyDnsDefaults('new-tunnel', ['geosite-netflix']);

    expect(api.singboxRouterAddDNSServer).not.toHaveBeenCalled();
    expect(api.singboxRouterUpdateDNSServer).toHaveBeenCalledWith(
      'dns-tunnel',
      expect.objectContaining({ tag: 'dns-tunnel', detour: 'new-tunnel' }),
    );
    expect(api.singboxRouterAddDNSRule).not.toHaveBeenCalled();
    expect(api.singboxRouterPutDNSGlobals).toHaveBeenCalledWith(
      expect.objectContaining({ final: 'dns-direct', strategy: 'ipv4_only' }),
    );
  });

  it('пустой ruleSetTags → DNS-правило не создаётся', async () => {
    (api.singboxRouterListDNSServers as ReturnType<typeof vi.fn>).mockResolvedValue([]);
    (api.singboxRouterListDNSRules as ReturnType<typeof vi.fn>).mockResolvedValue([]);
    (api.singboxRouterGetDNSGlobals as ReturnType<typeof vi.fn>).mockResolvedValue({ final: '', strategy: 'ipv4_only' });
    await applyDnsDefaults('my-selector', []);
    expect(api.singboxRouterAddDNSRule).not.toHaveBeenCalled();
  });
});
