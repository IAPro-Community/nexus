declare module 'node:fs' {
  export function readFileSync(path: string | URL, encoding: string): string;
}

declare module 'node:vm' {
  const vm: {
    runInNewContext(code: string, context: Record<string, unknown>): unknown;
  };

  export default vm;
}
