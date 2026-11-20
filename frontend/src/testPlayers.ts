import type { PlayerInfo } from "./types.ts";

/** Four connected test players named P0..P3; `powerUsed` flags by seat. */
export function makePlayers(powerUsed: readonly boolean[] = []): PlayerInfo[] {
  return [0, 1, 2, 3].map((i) => ({
    Name: `P${i}`,
    PowerUsed: powerUsed[i] ?? false,
    Connected: true,
  }));
}
