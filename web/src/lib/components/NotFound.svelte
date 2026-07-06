<script lang="ts">
	let { message, backHref, backLabel }: { message: string; backHref: string; backLabel: string } =
		$props();
</script>

<p>{message}</p>
<p><a href={backHref}>{backLabel}</a></p>
