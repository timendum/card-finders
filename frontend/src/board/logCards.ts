// Which cards a log entry is about, as (seat, slot) pairs, so the table can
// highlight them while the entry is hovered. Slots are positions in the hand of
// the seat that owns them.
import { JokerSwap, LogEvent, type LogEntry, type SeatID } from "../types.ts";

export interface SlotRef {
  seat: SeatID;
  slot: number;
}

// Go serializes an absent seat or slice as null, which the generated types miss.
function refs(seat: SeatID | null | undefined, slots: readonly number[] | null | undefined) {
  return seat == null ? [] : (slots ?? []).map((slot) => ({ seat, slot }));
}

export function logCardRefs(entry: LogEntry): SlotRef[] {
  switch (entry.Event) {
    case LogEvent.Call:
      // Called cards sit in the target's hand, the cards played from the actor's.
      return [...refs(entry.Target, entry.Slots), ...refs(entry.Actor, entry.Sources)];
    case LogEvent.Choice:
      return refs(entry.Actor, entry.Slots);
    case LogEvent.Joker: {
      const slots = entry.Slots ?? [];
      const after = entry.Sources ?? [];
      if (entry.Joker === JokerSwap && after.length === 2) {
        // Resolved swap: the swapped cards sit at their new positions, actor's first.
        return [
          ...refs(entry.Actor, [after[0] as number]),
          ...refs(entry.Target, [after[1] as number]),
        ];
      }
      return refs(entry.Actor, slots);
    }
    default:
      return [];
  }
}
