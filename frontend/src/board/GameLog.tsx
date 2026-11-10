// GameLog — scrollable list of public events with no visual clue (wrong calls,
// safe call uses, ...). Newest entry is at the bottom and kept in view.
import { useEffect, useRef } from "react";
import type { PlayerView } from "../types.ts";

// Position of the actor relative to the viewer, matching the table layout.
function actorName(viewer: number, actor: number): string {
  return ["You", "Left", "Top", "Right"][(actor - viewer + 4) % 4] ?? "You";
}

export function GameLog({ view }: { view: PlayerView }) {
  const endRef = useRef<HTMLLIElement>(null);
  const log = view.Log ?? [];

  useEffect(() => {
    endRef.current?.scrollIntoView?.({ block: "nearest" });
  }, [log.length]);

  return (
    <section className="game-log" aria-label="Game log">
      <ol className="game-log-list">
        {log.map((entry, i) => (
          <li key={`${entry.Seq}-${i}`} className="game-log-entry">
            {entry.HasActor && <strong>{actorName(view.Seat, entry.Actor)} </strong>}
            {entry.Text}
          </li>
        ))}
        <li ref={endRef} aria-hidden="true" />
      </ol>
    </section>
  );
}
