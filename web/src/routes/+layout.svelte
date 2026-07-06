<script lang="ts">
	import { onMount } from 'svelte';
	import favicon from '$lib/assets/favicon.png';

	let { children } = $props();

	onMount(async () => {
		const { playhtml } = await import('playhtml');
		playhtml.init({
			cursors: {
				enabled: true,
				getCursorStyle: () => ({ opacity: '0.5' })
			}
		});
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

{@render children()}
