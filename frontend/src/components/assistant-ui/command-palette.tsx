/** 
command-palette is a cosutomized component to show pop up command palette.
It defines SLASH_COMMANDS, and being triggered when query started with '/'.
*/

import { useRef, type FC } from "react";
import { TerminalIcon } from "lucide-react";

export interface SlashCommand {
  name: string;        // e.g. "/quit"
  description: string; // short label e.g. "Stop LLM processing"
  usage: string;       // e.g. "/quit"
}

export const SLASH_COMMANDS: SlashCommand[] = [
  {
    name: "/quit",
    description: "Stop LLM processing immediately",
    usage: "/quit",
  },
  {
    name: "/incident",
    description: "Show active incident or search team incidents by title/summary",
    usage: "/incident [query]",
  },
  {
    name: "/runbook",
    description: "List active team runbooks or search by title/content",
    usage: "/runbook [query]",
  },
];

interface CommandPaletteProps {
  query: string;           // current composer text
  onSelect: (cmd: SlashCommand) => void;
  onDismiss: () => void;
}

export const CommandPalette: FC<CommandPaletteProps> = ({ query, onSelect }) => {
  const ref = useRef<HTMLDivElement>(null);


  // only show when query starts with "/"
  const trimmed = query.trim().toLowerCase();
  if (!trimmed.startsWith("/")) return null;

  // hide palette if user has already selected/typed a command name followed by space
  if (SLASH_COMMANDS.some((cmd) => query.startsWith(cmd.name + " "))) {
    return null;
  }

  const matches = SLASH_COMMANDS.filter(
    (cmd) =>
      cmd.name.startsWith(trimmed) ||
      trimmed === "" ||
      trimmed === "/"
  );

  if (matches.length === 0) return null;

  return (
    <div
      ref={ref}
      className="command-palette"
      role="listbox"
      aria-label="Slash commands"
    >
      <div className="command-palette-header">
        <TerminalIcon className="command-palette-header-icon" />
        <span>Commands</span>
      </div>
      {matches.map((cmd) => (
        <button
          key={cmd.name}
          className="command-palette-item"
          role="option"
          aria-selected={false}
          onMouseDown={(e) => {
            // prevent blur on the textarea
            e.preventDefault();
            onSelect(cmd);
          }}
        >
          <div className="command-palette-item-left">
            <span className="command-palette-item-name">{cmd.name}</span>
            <span className="command-palette-item-usage">{cmd.usage}</span>
          </div>
          <span className="command-palette-item-desc">{cmd.description}</span>
        </button>
      ))}
      <div className="command-palette-footer">
        Press <kbd>Esc</kbd> to dismiss
      </div>
    </div>
  );
};
