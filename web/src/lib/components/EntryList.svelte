<script lang="ts" generics="T">
	import type { Snippet } from 'svelte';

	let {
		items,
		emptyMessage,
		key,
		item,
		class: className
	}: {
		items: T[];
		emptyMessage: string;
		key: (entry: T) => string | number | bigint;
		item: Snippet<[T]>;
		class?: string;
	} = $props();
</script>

{#if items.length === 0}
	<p>{emptyMessage}</p>
{:else}
	<ul class={className}>
		{#each items as entry (key(entry))}
			{@render item(entry)}
		{/each}
	</ul>
{/if}
