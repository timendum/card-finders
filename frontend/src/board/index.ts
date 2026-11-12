// Public surface of the board feature. Anything not exported here is internal
// to the folder; import it directly only from within board/.
export { Table } from "./Table.tsx";
export { PromptPanel } from "./PromptPanel.tsx";
export { GameLog } from "./GameLog.tsx";
export { matchHints } from "./matchHints.ts";
export {
  emptySelection,
  isSelectable,
  selectionReducer,
  type SelectionEvent,
  type SelectionState,
} from "./selection.ts";
export { messageFor } from "./confirm.ts";
