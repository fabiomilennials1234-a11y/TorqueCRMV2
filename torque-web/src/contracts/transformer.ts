type AnyObject = Record<string, unknown>;

function snakeToCamel(str: string): string {
  return str.replace(/_([a-z])/g, (_, c: string) => c.toUpperCase());
}

function camelToSnake(str: string): string {
  return str.replace(/[A-Z]/g, (c) => `_${c.toLowerCase()}`);
}

function transformKeys(obj: unknown, fn: (key: string) => string): unknown {
  if (Array.isArray(obj)) return obj.map((item) => transformKeys(item, fn));
  if (obj !== null && typeof obj === "object" && !(obj instanceof Date)) {
    return Object.fromEntries(
      Object.entries(obj as AnyObject).map(([key, value]) => [
        fn(key),
        transformKeys(value, fn),
      ]),
    );
  }
  return obj;
}

export function toClient<T>(serverData: unknown): T {
  return transformKeys(serverData, snakeToCamel) as T;
}

export function toServer(clientData: unknown): unknown {
  return transformKeys(clientData, camelToSnake);
}
