/** Whether a server connect host is empty (WAN fallback) or a valid IP/domain. */
export function isValidEndpointHost(val: string): boolean {
	const trimmed = val.trim();
	if (!trimmed) return true;
	if (/^(\d{1,3}\.){3}\d{1,3}$/.test(trimmed)) return true;
	if (/^([a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$/.test(trimmed)) return true;
	return false;
}
