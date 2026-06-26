export function AccessExplanation() {
  // Example: This would be dynamic in a real app
  const explanation =
    "Granted via JIT add-on for Project Phoenix, expires in 12h";
  return (
    <div className="mt-8 flex items-center gap-3 p-4 bg-yellow-50 border-l-4 border-yellow-400 rounded shadow">
      <span className="text-yellow-500 text-2xl">⚡</span>
      <span>
        <strong>Access Explanation:</strong> {explanation}
      </span>
    </div>
  );
}
