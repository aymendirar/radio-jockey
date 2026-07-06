<script lang="ts">
	import { onMount } from 'svelte';
	import favicon from '$lib/assets/favicon.png';
	import VisitorCount from '$lib/components/VisitorCount.svelte';

	let { children } = $props();
	let cursorViewport: HTMLDivElement;

	onMount(async () => {
		const { playhtml } = await import('playhtml');
		playhtml.init({
			// static (not path-derived) so the room spans every page — a sitewide visitor
			// count needs everyone in one room, not one room per route
			room: 'site',
			cursors: {
				enabled: true,
				enableChat: true,
				// mounting cursors in our own fixed+overflow:hidden container (instead of the
				// default document.body) clips any out-of-viewport cursor at the CSS level, so
				// there's no gap between a cursor going out of bounds and shouldRenderCursor
				// catching up on the next presence update (which showed up as a scrollbar flash)
				container: cursorViewport,
				// opacity gets clobbered by playhtml's own proximity fade logic on every
				// cursor update; filter isn't touched, so it's the only way to dim cursors
				getCursorStyle: () => ({ filter: 'opacity(0.5)' }),
				shouldRenderCursor: (presence) => {
					const cursor = presence.cursor;
					if (!cursor) return false;
					return (
						cursor.x >= 0 &&
						cursor.y >= 0 &&
						cursor.x <= window.innerWidth &&
						cursor.y <= window.innerHeight
					);
				}
			}
		});
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<div bind:this={cursorViewport} class="cursor-viewport"></div>

<h1>
	<a href="/" class="title-link"
		><span class="wave-group">
			<span class="wave-1">(</span> <span class="wave-2">(</span>
			<span class="wave-3">(</span>
		</span>
		<span class="title-text">radio jockey</span>
		<span class="wave-group">
			<span class="wave-3">)</span> <span class="wave-2">)</span>
			<span class="wave-1">)</span>
		</span></a
	>
	<VisitorCount />
</h1>

{@render children()}

<style>
	.cursor-viewport {
		position: fixed;
		inset: 0;
		overflow: hidden;
		pointer-events: none;
	}

	/* keeps the anchor's own box out of h1's flex layout so it doesn't
	   swallow VisitorCount into the clickable area */
	.title-link {
		display: contents;
	}
</style>
