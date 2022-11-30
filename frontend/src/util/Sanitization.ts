export default function EscapeHTML(unsafe: string): string {
	return unsafe.replace(
		/[\u0000-\u002F\u003A-\u0040\u005B-\u0060\u007B-\u00FF]/g,
		(c) => '&#' + ('000' + c.charCodeAt(0)).slice(-4) + ';'
	);
}
