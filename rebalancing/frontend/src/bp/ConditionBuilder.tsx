import React, { useState } from "react";

interface Props {
  mode?: "json";
  onChange: (val: string) => void;
}

export const ConditionBuilder: React.FC<Props> = ({ value, mode = "json", onChange }) => {
  const [editMode, setEditMode] = useState<"visual" | "code">(mode === "json" ? "visual" : "code");

  return (
    <div className="condition-builder border p-2 rounded">
      <div className="flex gap-2 mb-2">
        <button 
          className={`px-2 py-1 text-sm ${editMode === "visual" ? "bg-blue-100 font-bold" : "bg-gray-100"}`}
          onClick={() => setEditMode("visual")}
        >
          Visual
        </button>
        <button 
          className={`px-2 py-1 text-sm ${editMode === "code" ? "bg-blue-100 font-bold" : "bg-gray-100"}`}
          onClick={() => setEditMode("code")}
        >
          Code (Advanced)
        </button>
      </div>

      {editMode === "code" ? (
        <textarea
          className="w-full h-24 font-mono text-sm border p-1"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="e.g. num_field('req', 'amount') > 5000"
        />
      ) : (
        <div className="p-4 bg-gray-50 text-gray-500 text-sm italic">
          Visual Builder Placeholder (Tree View)
          {/* Implement Tree/Block builder here */}
        </div>
      )}
    </div>
  );
};
