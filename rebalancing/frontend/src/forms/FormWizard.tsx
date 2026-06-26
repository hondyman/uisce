import React, { useMemo } from "react";
import { usePersistedForm } from "../hooks/usePersistedForm";
import { DynamicForm } from "./DynamicForm";
import { FormSchema } from "./formSchema";
import { evaluateCondition } from "./formLogic"; // Assuming eval logic is exported or mocked

// Defines the state shape for a multi-step wizard
type WizardState = {
  currentStepKey: string;
  valuesByStep: Record<string, Record<string, any>>;
};

interface WizardStep {
  key: string;
  label: string;
  schema: FormSchema;
}

interface Props {
  storageKey: string;
  steps: WizardStep[];
  userRoles?: string[];
  onComplete?: (data: Record<string, any>) => void;
}

const initialWizardState = (firstStep: string): WizardState => ({
  currentStepKey: firstStep,
  valuesByStep: {},
});

export const FormWizard: React.FC<Props> = ({ 
  storageKey, 
  steps, 
  userRoles = [],
  onComplete 
}) => {
  const [wizard, setWizard, clearWizard] = usePersistedForm<WizardState>(
    storageKey,
    initialWizardState(steps[0]?.key || "start"),
    "session"
  );

  const currentStepIndex = steps.findIndex(s => s.key === wizard.currentStepKey);
  const currentStep = steps[currentStepIndex];
  const currentValues = wizard.valuesByStep[wizard.currentStepKey] || {};

  // Simple mock evaluator for now, real one would use RuleEngine
  const evalCondition = (cond: any, values: any) => true; 

  const updateField = (key: string, value: any) => {
    setWizard(prev => ({
      ...prev,
      valuesByStep: {
        ...prev.valuesByStep,
        [prev.currentStepKey]: {
          ...(prev.valuesByStep[prev.currentStepKey] || {}),
          [key]: value
        }
      }
    }));
  };

  const goToStep = (stepKey: string) => {
    setWizard(prev => ({ ...prev, currentStepKey: stepKey }));
  };

  const handleNext = () => {
    const nextIndex = currentStepIndex + 1;
    if (nextIndex < steps.length) {
      goToStep(steps[nextIndex].key);
    } else {
      // Complete
      const allData = Object.values(wizard.valuesByStep).reduce((acc, v) => ({ ...acc, ...v }), {});
      if (onComplete) onComplete(allData);
      clearWizard(); // Optional: clear on success
    }
  };

  const handleBack = () => {
    const prevIndex = currentStepIndex - 1;
    if (prevIndex >= 0) {
      goToStep(steps[prevIndex].key);
    }
  };

  if (!currentStep) return <div>No steps defined</div>;

  return (
    <div className="wizard-container flex flex-col h-full">
      {/* Progress Bar */}
      <div className="wizard-steps flex border-b bg-gray-50 p-2 overflow-x-auto space-x-2">
        {steps.map((s, idx) => {
          const isActive = s.key === wizard.currentStepKey;
          const isPast = idx < currentStepIndex;
          return (
            <div 
              key={s.key} 
              className={`px-3 py-1 rounded text-sm cursor-pointer whitespace-nowrap
                ${isActive ? 'bg-blue-600 text-white font-bold' : 
                  isPast ? 'bg-blue-100 text-blue-800' : 'bg-gray-200 text-gray-500'}`}
              onClick={() => isPast && goToStep(s.key)}
            >
              {idx + 1}. {s.label}
            </div>
          );
        })}
      </div>

      {/* Form Area */}
      <div className="flex-grow p-6 overflow-y-auto">
        <h2 className="text-xl font-bold mb-4">{currentStep.label}</h2>
        <DynamicForm
          schema={currentStep.schema}
          values={currentValues}
          onChange={updateField}
          userRoles={userRoles}
          evalConditionJson={evalCondition}
        />
      </div>

      {/* Actions */}
      <div className="wizard-actions border-t p-4 flex justify-between bg-white">
        <button 
          onClick={handleBack} 
          disabled={currentStepIndex === 0}
          className="px-4 py-2 border rounded text-sm disabled:opacity-50"
        >
          Back
        </button>
        <div className="space-x-2">
           <button 
             onClick={() => clearWizard()}
             className="px-4 py-2 text-red-500 text-sm hover:underline"
           >
             Reset Form
           </button>
           <button 
             onClick={handleNext}
             className="px-4 py-2 bg-blue-600 text-white rounded text-sm hover:bg-blue-700"
           >
             {currentStepIndex === steps.length - 1 ? 'Submit' : 'Next'}
           </button>
        </div>
      </div>
    </div>
  );
};
