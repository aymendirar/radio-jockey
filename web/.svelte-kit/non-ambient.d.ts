// this file is generated — do not edit it

declare module 'svelte/elements' {
	export interface HTMLAttributes<T> {
		'data-sveltekit-keepfocus'?: true | '' | 'off' | undefined | null;
		'data-sveltekit-noscroll'?: true | '' | 'off' | undefined | null;
		'data-sveltekit-preload-code'?:
			| true
			| ''
			| 'eager'
			| 'viewport'
			| 'hover'
			| 'tap'
			| 'off'
			| undefined
			| null;
		'data-sveltekit-preload-data'?: true | '' | 'hover' | 'tap' | 'off' | undefined | null;
		'data-sveltekit-reload'?: true | '' | 'off' | undefined | null;
		'data-sveltekit-replacestate'?: true | '' | 'off' | undefined | null;
	}
}

export {};

declare module '$app/types' {
	type MatcherParam<M> = M extends ((param: string) => param is infer U extends string)
		? U
		: string;

	export interface AppTypes {
		RouteId(): '/' | '/radio' | '/radio/create' | '/radio/listen' | '/radio/listen/[sessionId]';
		RouteParams(): {
			'/radio/listen/[sessionId]': { sessionId: string };
		};
		LayoutParams(): {
			'/': { sessionId?: string };
			'/radio': { sessionId?: string };
			'/radio/create': Record<string, never>;
			'/radio/listen': { sessionId?: string };
			'/radio/listen/[sessionId]': { sessionId: string };
		};
		Pathname():
			| '/'
			| '/radio'
			| '/radio/create'
			| '/radio/listen'
			| (`/radio/listen/${string}` & {});
		ResolvedPathname(): `${'' | `/${string}`}${ReturnType<AppTypes['Pathname']>}`;
		Asset(): '/robots.txt' | (string & {});
	}
}
