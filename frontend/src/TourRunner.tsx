import { useState, useEffect, useLayoutEffect, useMemo } from 'react';
import { devError } from './utils/devLogger';
import { getTour } from './api';
import type { TourStep } from './types';

interface TourRunnerProps {
  tourId: string;
  onComplete: () => void;
}

function TourStepTooltip({ step, totalSteps, onNext, onPrev, onEnd }: { step: TourStep, totalSteps: number, onNext: () => void, onPrev: () => void, onEnd: () => void }) {
  const instanceId = useMemo(() => Math.random().toString(36).slice(2,9), []);

  useLayoutEffect(() => {
    const targetElement = document.querySelector(step.target_selector);
    const styleId = `tour-tooltip-style-${instanceId}`;
    let styleEl: HTMLStyleElement | null = document.getElementById(styleId) as HTMLStyleElement | null;
    if (!styleEl) {
      styleEl = document.createElement('style');
      styleEl.id = styleId;
      document.head.appendChild(styleEl);
    }

    if (targetElement && styleEl) {
      const rect = targetElement.getBoundingClientRect();
      targetElement.classList.add('tour-highlight');

      let css = '';
      if (step.position === 'bottom') {
        css = `.${styleId} { top: ${rect.bottom + 10}px; left: ${rect.left}px; }`;
      } else if (step.position === 'right') {
        css = `.${styleId} { top: ${rect.top}px; left: ${rect.right + 10}px; }`;
      } else {
        css = `.${styleId} { top: ${rect.bottom + 10}px; left: ${rect.left}px; }`;
      }
      styleEl.textContent = css;
    }

    return () => {
      if (targetElement) targetElement.classList.remove('tour-highlight');
      const el = document.getElementById(`tour-tooltip-style-${instanceId}`);
      if (el && el.parentNode) el.parentNode.removeChild(el);
    };
  }, [step, instanceId]);

  return (
    <div className={`tour-tooltip tour-tooltip-style-${instanceId}`}>
      <h4>{step.title} ({step.step}/{totalSteps})</h4>
      <p>{step.content}</p>
      <div className="tour-actions">
        <button onClick={onEnd}>End Tour</button>
        <button onClick={onPrev} disabled={step.step === 1}>Prev</button>
        <button onClick={onNext}>{step.step === totalSteps ? 'Finish' : 'Next'}</button>
      </div>
    </div>
  );
}

export default function TourRunner({ tourId, onComplete }: TourRunnerProps) {
  const [steps, setSteps] = useState<TourStep[]>([]);
  const [currentStepIndex, setCurrentStepIndex] = useState(0);

  useEffect(() => {
    getTour(tourId).then(data => setSteps(data.steps)).catch((e) => { devError(e); });
  }, [tourId]);

  const handleNext = () => {
    if (currentStepIndex < steps.length - 1) {
      setCurrentStepIndex(currentStepIndex + 1);
    } else {
      onComplete();
    }
  };

  const handlePrev = () => {
    if (currentStepIndex > 0) {
      setCurrentStepIndex(currentStepIndex - 1);
    }
  };

  const currentStep = steps[currentStepIndex];
  if (!currentStep) return null;

  return <TourStepTooltip step={currentStep} totalSteps={steps.length} onNext={handleNext} onPrev={handlePrev} onEnd={onComplete} />;
}