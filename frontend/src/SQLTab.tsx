// React default import not required

export default function SQLTab({ sql }: { sql?: string }) {
  return (
    <div className="sql-tab">
      <button onClick={() => navigator.clipboard.writeText(sql || '')}>Copy SQL</button>
      <pre>{sql}</pre>
    </div>
  );
}