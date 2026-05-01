type Metadata = Record<string, unknown>;

function format(level: string, message: string, metadata?: Metadata): string {
  const entries = Object.entries(metadata ?? {});
  const meta =
    entries.length > 0
      ? " {" + entries.map(([k, v]) => ` ${k}: ${v}`).join(",") + " }"
      : "";
  return `[discord-jockey] ${level}: ${message}${meta}`;
}

export const logger = {
  info(message: string, metadata?: Metadata): void {
    console.log(format("INFO", message, metadata));
  },
  error(message: string, metadata?: Metadata): void {
    console.error(format("ERROR", message, metadata));
  },
  warn(message: string, metadata?: Metadata): void {
    console.warn(format("WARN", message, metadata));
  },
};
