
// this file is generated — do not edit it


declare module "svelte/elements" {
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


declare module "$app/types" {
	type MatcherParam<M> = M extends (param : string) => param is (infer U extends string) ? U : string;

	export interface AppTypes {
		RouteId(): "/" | "/admin" | "/archive" | "/archive/[archiveId]" | "/stations" | "/stations/create" | "/stations/[sessionId]";
		RouteParams(): {
			"/archive/[archiveId]": { archiveId: string };
			"/stations/[sessionId]": { sessionId: string }
		};
		LayoutParams(): {
			"/": { archiveId?: string; sessionId?: string };
			"/admin": Record<string, never>;
			"/archive": { archiveId?: string };
			"/archive/[archiveId]": { archiveId: string };
			"/stations": { sessionId?: string };
			"/stations/create": Record<string, never>;
			"/stations/[sessionId]": { sessionId: string }
		};
		Pathname(): "/" | "/admin" | "/archive" | `/archive/${string}` & {} | "/stations" | "/stations/create" | `/stations/${string}` & {};
		ResolvedPathname(): `${"" | `/${string}`}${ReturnType<AppTypes['Pathname']>}`;
		Asset(): "/radio.png" | "/robots.txt" | string & {};
	}
}