
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
		RouteId(): "/" | "/admin" | "/stations" | "/stations/create" | "/stations/[sessionId]";
		RouteParams(): {
			"/stations/[sessionId]": { sessionId: string }
		};
		LayoutParams(): {
			"/": { sessionId?: string };
			"/admin": Record<string, never>;
			"/stations": { sessionId?: string };
			"/stations/create": Record<string, never>;
			"/stations/[sessionId]": { sessionId: string }
		};
		Pathname(): "/" | "/admin" | "/stations" | "/stations/create" | `/stations/${string}` & {};
		ResolvedPathname(): `${"" | `/${string}`}${ReturnType<AppTypes['Pathname']>}`;
		Asset(): "/.DS_Store" | "/04B_30.TTF" | "/04b_30/.DS_Store" | "/04b_30/about.gif" | "/04b_30.zip" | "/Pixeled.ttf" | "/dogica/Ascii GB Studio/01.png" | "/dogica/Ascii GB Studio/02.png" | "/dogica/Ascii GB Studio/03.png" | "/dogica/Ascii GB Studio/04.png" | "/dogica/OTF/dogica.otf" | "/dogica/OTF/dogicabold.otf" | "/dogica/OTF/dogicapixel.otf" | "/dogica/OTF/dogicapixelbold.otf" | "/dogica/Specimen/dogica.png" | "/dogica/Specimen/dogicabold.png" | "/dogica/TTF/.DS_Store" | "/dogica/TTF/dogicabold.ttf" | "/dogica/TTF/dogicapixel.ttf" | "/dogica/TTF/dogicapixelbold.ttf" | "/dogica/dogica_license.txt" | "/dogica/dogica_pixel_license.txt" | "/dogica/info.txt" | "/dogica.ttf" | "/dogica.zip" | "/pixel.ttf" | "/pixel_3/.DS_Store" | "/pixel_3/license.txt" | "/pixel_3/readme.txt" | "/pixel_3.zip" | "/radio.png" | "/radio2.png" | "/robots.txt" | "/windows-xp-tahoma.otf.zip" | string & {};
	}
}