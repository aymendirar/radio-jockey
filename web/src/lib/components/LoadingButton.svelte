<script lang="ts">
	let {
		loading,
		label,
		onclick,
		type = 'button'
	}: {
		loading: boolean;
		label: string;
		onclick?: () => void;
		type?: 'button' | 'submit';
	} = $props();
</script>

<button {type} {onclick} disabled={loading}>{loading ? 'loading...' : label}</button>
