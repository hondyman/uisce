import { useState } from 'react';
import { JITRequestPanel } from "./JITRequestPanel";

export function JITRequestPanelExample() {
  const [open, setOpen] = useState(true);
  return (
    <div>
      {open && <JITRequestPanel onClose={() => setOpen(false)} />}
      {!open && (
        <button className="bg-blue-600 text-white px-4 py-2 rounded" onClick={() => setOpen(true)}>
          Open JIT Request Panel
        </button>
      )}
    </div>
  );
}
