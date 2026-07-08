<script lang="ts">
	let {
		loading,
		label,
		onclick,
		type = 'button',
		disabled = false
	}: {
		loading: boolean;
		label: string;
		onclick?: () => void;
		type?: 'button' | 'submit';
		disabled?: boolean;
	} = $props();
</script>

<button {type} {onclick} disabled={loading || disabled}>{loading ? 'loading...' : label}</button>
