<script lang="ts">
	import { api } from '$lib/api/client';
	import type { GeoFileEntry } from '$lib/types';
	import { HrNeoGeoDataView } from '$lib/components/hrneo';

	let geoFiles = $state<GeoFileEntry[]>([]);

	async function loadGeoFiles() {
		try {
			geoFiles = (await api.getGeoFiles()) ?? [];
		} catch {
			geoFiles = [];
		}
	}

	$effect(() => {
		void loadGeoFiles();
	});
</script>

<HrNeoGeoDataView files={geoFiles} onrefresh={loadGeoFiles} />
