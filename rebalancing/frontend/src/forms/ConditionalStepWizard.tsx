import React, { useMemo } from "react";
import { RuleAwareStepForm } from "./RuleAwareStepForm";
import { WizardStepDef } from "./ruleAwareFormTypes";

interface Props {
  steps: WizardStepDef[];
  userRoles: string[];
  evalConditionJson: (cond: any, values: Record<string, any>) => boolean;
  onComplete: (allValues: Record<string, any>) => Promise<void>;
}

export const ConditionalStepWizard: React.FC<Props> = ({
  steps,
  userRoles,
  evalConditionJson,
  onComplete,
}) => {
  const [currentStepKey, setCurrentStepKey] = React.useState(steps[0]?.key);
  const [allValues, setAllValues] = React.useState<Record<string, any>>({});
  const [history, setHistory] = React.useState<string[]>([steps[0]?.key]); // Track navigation history

  const currentStep = useMemo(
    () => steps.find((s) => s.key === currentStepKey),
    [currentStepKey, steps]
  );

  const handleStepSubmit = async (stepValues: Record<string, any>) => {
    // Merge step values
    const newAllValues = { ...allValues, ...stepValues };
    setAllValues(newAllValues);

    if (!currentStep) return;

    // Resolve next step via branching conditions
    let nextStepKey = currentStep.defaultNextStep;
    if (currentStep.branches) {
      for (const branch of currentStep.branches) {
        try {
          if (evalConditionJson(branch.condition, newAllValues)) {
            nextStepKey = branch.nextStepKey;
            break;
          }
        } catch (e) {
          console.error(`Branch condition failed:`, e);
        }
      }
    }

    if (!nextStepKey) {
      // Workflow complete
      await onComplete(newAllValues);
      return;
    }

    setHistory([...history, nextStepKey]);
    setCurrentStepKey(nextStepKey);
  };

  const handleBack = () => {
      if (history.length > 1) {
          const newHistory = [...history];
          newHistory.pop(); // Remove current
          const prevStep = newHistory[newHistory.length - 1];
          setHistory(newHistory);
          setCurrentStepKey(prevStep);
      }
  };

  if (!currentStep) return <div>No steps available</div>;

  const currentStepIndex = steps.findIndex(s => s.key === currentStepKey);

  return (
    <div className="max-w-3xl mx-auto p-6 bg-gray-50 min-h-screen">
      <div className="mb-8">
        <h2 className="text-2xl font-bold text-gray-900">{currentStep.label}</h2>
        <div className="mt-2 h-2 bg-gray-200 rounded-full overflow-hidden">
             {/* Progress bar approximation */}
             <div 
                className="h-full bg-blue-600 transition-all duration-300"
                style={{ width: `${Math.min(100, ((history.length) / steps.length) * 100)}%` }}
             />
        </div>
        <p className="text-sm text-gray-500 mt-1">
          Step {history.length}
        </p>
      </div>

      <div className="bg-white rounded-lg shadow-sm p-6 border">
        <RuleAwareStepForm
            key={currentStepKey} // Force reset on step change
            schema={currentStep.schema}
            initialValues={allValues}
            userRoles={userRoles}
            evalConditionJson={evalConditionJson}
            onSubmit={handleStepSubmit}
        />
      </div>

      <div className="mt-4 flex justify-between">
        <button
          onClick={handleBack}
          disabled={history.length <= 1}
          className={`px-4 py-2 rounded text-sm font-medium ${
            history.length <= 1 
                ? "bg-gray-100 text-gray-400 cursor-not-allowed" 
                : "bg-white border text-gray-700 hover:bg-gray-50"
          }`}
        >
          Back
        </button>
      </div>
    </div>
  );
};
