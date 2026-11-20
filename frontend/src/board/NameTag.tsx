import { useState } from "react";

export interface NameTagProps {
  name: string;
  // Present only for the viewer's own tag; makes it editable.
  onRename?: (name: string) => void;
}

const MAX_NAME_LENGTH = 20;

export function NameTag({ name, onRename }: NameTagProps) {
  // The text being typed; null while not editing.
  const [draft, setDraft] = useState<string | null>(null);

  if (draft === null) {
    return onRename ? (
      <button
        type="button"
        className="name-tag"
        title="Change your name"
        onClick={() => setDraft(name)}
      >
        {name}
      </button>
    ) : (
      <span className="name-tag">{name}</span>
    );
  }

  const commit = () => {
    const next = draft.trim();
    if (next && next !== name) onRename?.(next);
    setDraft(null);
  };

  return (
    <input
      className="name-tag name-tag-input"
      aria-label="Your name"
      value={draft}
      maxLength={MAX_NAME_LENGTH}
      autoFocus
      onChange={(e) => setDraft(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => {
        if (e.key === "Enter") commit();
        else if (e.key === "Escape") setDraft(null);
      }}
    />
  );
}
