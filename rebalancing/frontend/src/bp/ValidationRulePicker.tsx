import React from "react";

interface Props {
  selectedRuleIds: string[];
  onChange: (ids: string[]) => void;
}

export const ValidationRulePicker: React.FC<Props> = ({ selectedRuleIds = [], onChange }) => {
  // Mock available rules
  const availableRules = [
    { id: "rule-amt-limit", name: "Amount Limit Check" },
    { id: "rule-kyc-check", name: "KYC Verification" },
    { id: "rule-risk-score", name: "Risk Score Check" },
  ];

  const toggleRule = (id: string) => {
    if (selectedRuleIds.includes(id)) {
      onChange(selectedRuleIds.filter((r) => r !== id));
    } else {
      onChange([...selectedRuleIds, id]);
    }
  };

  return (
    <div className="border p-2 rounded bg-gray-50 max-h-40 overflow-y-auto">
      {availableRules.map((rule) => (
        <label key={rule.id} className="flex items-center gap-2 text-sm p-1 hover:bg-gray-100 cursor-pointer">
          <input
            type="checkbox"
            checked={selectedRuleIds.includes(rule.id)}
            onChange={() => toggleRule(rule.id)}
          />
          <span>{rule.name} <span className="text-gray-400 text-xs">({rule.id})</span></span>
        </label>
      ))}
    </div>
  );
};
