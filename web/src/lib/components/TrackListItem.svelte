<script lang="ts">
	import type { Track } from '$lib/proto/radio-jockey_pb';
	import type { Snippet } from 'svelte';

	let {
		track,
		href,
		children
	}: { track: Track; href?: string; children?: Snippet } = $props();
</script>

<li>
	{#if track.albumArtUrl}
		<img class="album-art-small" src={track.albumArtUrl} alt="" />
	{/if}
	{#if href}
		<a {href} target="_blank">{track.title} — {track.artist}</a>
	{:else}
		{track.title} — {track.artist}
	{/if}
	{#if children}
		{@render children()}
	{/if}
</li>

<style>
	.album-art-small {
		width: 40px;
		height: 30px;
		object-fit: cover;
		border: 1px solid white;
		vertical-align: middle;
		margin-right: 6px;
	}
</style>
