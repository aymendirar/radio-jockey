<script lang="ts">
	import LoadingButton from './LoadingButton.svelte';

	let { onSearch, searching }: { onSearch: (query: string) => void; searching: boolean } = $props();

	let query = $state('');

	function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (!query.trim()) return;
		onSearch(query.trim());
	}
</script>

<form onsubmit={handleSubmit}>
	<label for="ytQuery">query</label>
	<input
		id="ytQuery"
		type="text"
		bind:value={query}
		disabled={searching}
		placeholder="song or artist"
	/>
	<LoadingButton type="submit" loading={searching} label="search" />
</form>
