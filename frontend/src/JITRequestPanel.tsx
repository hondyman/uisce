import { useState } from "react";
import { useNotification } from './hooks/useNotification';

export function JITRequestPanel({ onClose }: { onClose: () => void }) {
  const [riskScore, _setRiskScore] = useState("Low");
  const [expiresIn, _setExpiresIn] = useState(4); // hours
  const [policyCheck, _setPolicyCheck] = useState("Pass");
  const [renewalRequested, setRenewalRequested] = useState(false);
  const [preApproved, _setPreApproved] = useState(true);
  const notification = useNotification();

  return (
    <div className="fixed inset-0 bg-black bg-opacity-40 flex items-center justify-center z-50 transition-opacity duration-200">
      <div className="bg-white p-8 rounded-lg shadow-2xl max-w-md w-full relative animate-fade-in">
        <button
          className="absolute top-2 right-2 text-gray-400 hover:text-gray-700 text-2xl"
          onClick={onClose}
          aria-label="Close"
        >
          ×
        </button>
        <h3 className="text-2xl font-bold mb-4">JIT Add-On Request</h3>
        <div className="mb-3 flex items-center gap-2">
          <strong>Risk Score:</strong> <span className="text-green-600 font-semibold">{riskScore}</span>
        </div>
        <div className="mb-3 flex items-center gap-2">
          <strong>Policy Check:</strong> <span className="text-green-600 font-semibold">{policyCheck}</span>
        </div>
        <div className="mb-3 flex items-center gap-2">
          <strong>Expires in:</strong> <span className="text-blue-600 font-semibold">{expiresIn} hours</span>
        </div>
        {preApproved ? (
          <button
            className="bg-blue-600 hover:bg-blue-700 text-white px-5 py-2 rounded mt-4 font-semibold shadow"
            onClick={() => notification.success('JIT Grant Approved!')}
          >
            One-Click Grant
          </button>
        ) : (
          <button
            className="bg-gray-400 text-white px-5 py-2 rounded mt-4 font-semibold cursor-not-allowed"
            disabled
          >
            Approval Required
          </button>
        )}
        <div className="mt-6">
          <button
            className="text-blue-600 underline hover:text-blue-800"
            onClick={() => setRenewalRequested(true)}
            disabled={renewalRequested}
          >
            {renewalRequested ? "Renewal Requested" : "Request Renewal"}
          </button>
        </div>
        <div className="mt-6 text-sm text-gray-600 italic">
          Auto-suggested based on recent denied actions.
        </div>
      </div>
    </div>
  );
}
