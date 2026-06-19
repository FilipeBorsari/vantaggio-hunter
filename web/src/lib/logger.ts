const isDev = process.env.NODE_ENV === "development";

type LogCtx = Record<string, unknown>;

function emit(level: string, msg: string, ctx?: LogCtx) {
  const line = JSON.stringify({ level, msg, ...ctx });
  if (level === "ERROR") return console.error(line);
  if (level === "WARN") return console.warn(line);
  if (level === "DEBUG") return console.debug(line);
  console.info(line);
}

export const logger = {
  info: (msg: string, ctx?: LogCtx) => emit("INFO", msg, ctx),
  warn: (msg: string, ctx?: LogCtx) => emit("WARN", msg, ctx),
  error: (msg: string, ctx?: LogCtx) => emit("ERROR", msg, ctx),
  debug: (msg: string, ctx?: LogCtx) => { if (isDev) emit("DEBUG", msg, ctx); },
};
