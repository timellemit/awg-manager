import { describe, expect, it } from 'vitest';
import { highlightJson } from './shareEditorHighlight';

describe('highlightJson', () => {
	it('colors object keys blue and string values amber', () => {
		const html = highlightJson(
			'[\n  {\n    "domain_keyword": [\n      "youtube"\n    ]\n  }\n]',
		);
		expect(html).toContain('hl-json-key">"domain_keyword"</span>');
		expect(html).toContain('hl-json-str">"youtube"</span>');
		expect(html).not.toContain('hl-json-key">"youtube"');
	});
});
