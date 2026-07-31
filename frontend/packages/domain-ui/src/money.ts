// Money helpers. All amounts are integer fen (分); floating point is forbidden.

function assertIntegerFen(value: number, name: string): void {
  if (!Number.isInteger(value)) {
    throw new Error(`${name} must be an integer amount of fen, got: ${value}`);
  }
}

/**
 * Format an integer fen amount as a yuan string, e.g. 1250 -> "¥12.50".
 */
export function formatFen(fen: number): string {
  assertIntegerFen(fen, "fen");
  const sign = fen < 0 ? "-" : "";
  const abs = Math.abs(fen);
  const yuan = Math.floor(abs / 100);
  const remainder = abs % 100;
  return `${sign}¥${yuan}.${String(remainder).padStart(2, "0")}`;
}

/**
 * Add two integer fen amounts. Rejects non-integer operands.
 */
export function addFen(a: number, b: number): number {
  assertIntegerFen(a, "a");
  assertIntegerFen(b, "b");
  return a + b;
}
