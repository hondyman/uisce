import { useState } from 'react';

export function StewardshipModal({
  scriptId, currentSteward, onClose, onSaved
}: {
  scriptId: string;
  currentSteward?: string;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [steward, setSteward] = useState(currentSteward || '');
  const [team, setTeam] = useState('');
  const [notes, setNotes] = useState('');

  const save = async () => {
    await fetch(`/api/scripts/${scriptId}/stewardship`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ steward, team, notes })
    });
    onSaved();
  };

  return (
    <div className="modal">
      <h3>Assign stewardship</h3>
      <label>Steward</label>
      <input value={steward} onChange={e => setSteward(e.target.value)} placeholder="user or team" />
      <label>Team</label>
      <input value={team} onChange={e => setTeam(e.target.value)} placeholder="optional team name" />
      <label>Notes</label>
      <textarea value={notes} onChange={e => setNotes(e.target.value)} placeholder="change rationale" />
      <div className="actions">
        <button onClick={save}>Save</button>
        <button onClick={onClose}>Cancel</button>
      </div>
    </div>
  );
}
