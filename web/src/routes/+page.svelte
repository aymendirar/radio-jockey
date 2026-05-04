<script lang="ts">
	import { onMount } from 'svelte';
	import { radioClient } from '$lib/connect/client';

	let message = $state('Pinging server...');
	let error = $state('');

	onMount(async () => {
		try {
			const response = await radioClient.ping({});
			message = response.message;
		} catch (err) {
			message = 'Ping failed';
			error = err instanceof Error ? err.message : String(err);
		}
	});
</script>

<div>
	<ul>
		<a href="/stations/create"><li>create a new station</li></a>
	</ul>
	<ul>
		<a href="/stations"><li>listen to live stations</li></a>
	</ul>
</div>
<p>{message}</p>
{#if error}
	<p>{error}</p>
{/if}

<style>
	ul {
		list-style-type: '>  ';
	}
</style>
