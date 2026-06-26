import React, { useState, useEffect } from 'react';
import { ChevronRight, ChevronLeft, Check, Upload, FileText, Shield } from 'lucide-react';

interface OnboardingWizardProps {
  onComplete: (data: OnboardingData) => void;
}

interface OnboardingData {
  personalInfo: PersonalInfoData;
  employment: EmploymentData;
  goals: GoalsData;
  riskAssessment: RiskAssessmentData;
  documents: UploadedDocument[];
  accountFunding: AccountFundingData;
  signatures: SignatureData[];
}

interface PersonalInfoData {
  firstName: string;
  middleName?: string;
  lastName: string;
  dateOfBirth: string;
  ssn: string;
  phone: string;
  email: string;
  addressLine1: string;
  addressLine2?: string;
  city: string;
  state: string;
  zipCode: string;
  country: string;
  citizenshipStatus: string;
}

interface EmploymentData {
  employmentStatus: string;
  employer?: string;
  occupation?: string;
  annualIncome: number;
  netWorth?: number;
  liquidNetWorth?: number;
}

interface GoalsData {
  primaryGoal: string;
  timeHorizon: number;
  additionalGoals: string[];
}

interface RiskAssessmentData {
  investmentExperience: string;
  riskTolerance: string;
  timeHorizon: number;
  liquidityNeeds: string;
  questionnaireAnswers: Record<string, any>;
}

interface UploadedDocument {
  documentId: string;
  documentType: string;
  fileName: string;
  fileUrl: string;
  verificationStatus: string;
}

interface AccountFundingData {
  accountTypes: string[];
  initialFunding: number;
  transferFrom?: string;
}

interface SignatureData {
  signatureId: string;
  documentName: string;
  status: string;
  signedAt?: Date;
}

const STEPS = [
  { number: 1, name: 'Personal Information', icon: Shield },
  { number: 2, name: 'Employment & Income', icon: FileText },
  { number: 3, name: 'Financial Goals', icon: Check },
  { number: 4, name: 'Risk Assessment', icon: Shield },
  { number: 5, name: 'Documents', icon: Upload },
  { number: 6, name: 'Account Setup', icon: FileText },
  { number: 7, name: 'Signatures', icon: Check },
];

export const OnboardingWizard: React.FC<OnboardingWizardProps> = ({ onComplete }) => {
  const [currentStep, setCurrentStep] = useState(1);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [formData, setFormData] = useState<Partial<OnboardingData>>({});
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [isSaving, setIsSaving] = useState(false);

  // Auto-save every 30 seconds
  useEffect(() => {
    const interval = setInterval(() => {
      if (sessionId) {
        saveProgress();
      }
    }, 30000);

    return () => clearInterval(interval);
  }, [sessionId, formData]);

  // Initialize or resume session
  useEffect(() => {
    const initSession = async () => {
      // Check for existing session in localStorage
      const existingSessionId = localStorage.getItem('onboarding_session_id');
      
      if (existingSessionId) {
        // Resume session
        const response = await fetch(`/api/onboarding/sessions/${existingSessionId}`);
        if (response.ok) {
          const session = await response.json();
          setSessionId(existingSessionId);
          setCurrentStep(session.current_step);
          setFormData(session.step_data);
          return;
        }
      }

      // Start new session
      const response = await fetch('/api/onboarding/sessions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: localStorage.getItem('user_email') || '',
        }),
      });

      if (response.ok) {
        const session = await response.json();
        setSessionId(session.session_id);
        localStorage.setItem('onboarding_session_id', session.session_id);
      }
    };

    initSession();
  }, []);

  const saveProgress = async () => {
    if (!sessionId) return;

    setIsSaving(true);
    try {
      await fetch(`/api/onboarding/sessions/${sessionId}/save`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ stepData: formData }),
      });
    } catch (error) {
      console.error('Failed to save progress:', error);
    } finally {
      setIsSaving(false);
    }
  };

  const validateStep = async (step: number): Promise<boolean> => {
    const newErrors: Record<string, string> = {};

    switch (step) {
      case 1: // Personal Info
        const personalInfo = formData.personalInfo;
        if (!personalInfo) return false;
        
        if (!personalInfo.firstName) newErrors.firstName = 'First name is required';
        if (!personalInfo.lastName) newErrors.lastName = 'Last name is required';
        if (!personalInfo.email) newErrors.email = 'Email is required';
        if (!personalInfo.ssn || personalInfo.ssn.length !== 11) {
          newErrors.ssn = 'Valid SSN required (XXX-XX-XXXX)';
        }
        
        // Server-side validation
        const response = await fetch('/api/onboarding/validate/personal-info', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(personalInfo),
        });

        if (!response.ok) {
          const error = await response.json();
          Object.assign(newErrors, error.errors);
        }
        break;

      case 2: // Employment
        const employment = formData.employment;
        if (!employment) return false;
        
        if (employment.employmentStatus === 'EMPLOYED' && !employment.employer) {
          newErrors.employer = 'Employer is required for employed individuals';
        }
        if (!employment.annualIncome || employment.annualIncome < 0) {
          newErrors.annualIncome = 'Valid annual income is required';
        }
        break;

      case 3: // Goals
        if (!formData.goals?.primaryGoal) {
          newErrors.primaryGoal = 'Please select a primary financial goal';
        }
        break;

      case 4: // Risk Assessment
        if (!formData.riskAssessment?.riskTolerance) {
          newErrors.riskTolerance = 'Please complete the risk assessment';
        }
        break;

      case 5: // Documents
        if (!formData.documents || formData.documents.length === 0) {
          newErrors.documents = 'Please upload required identification documents';
        }
        break;
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const nextStep = async () => {
    const isValid = await validateStep(currentStep);
    if (!isValid) return;

    await saveProgress();

    if (currentStep < 7) {
      setCurrentStep(currentStep + 1);
      
      // Update server with new step
      if (sessionId) {
        await fetch(`/api/onboarding/sessions/${sessionId}/step`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ step: currentStep + 1, stepData: formData }),
        });
      }
    } else {
      // Complete onboarding
      await completeOnboarding();
    }
  };

  const prevStep = () => {
    if (currentStep > 1) {
      setCurrentStep(currentStep - 1);
    }
  };

  const completeOnboarding = async () => {
    if (!sessionId) return;

    try {
      await fetch(`/api/onboarding/sessions/${sessionId}/complete`, {
        method: 'POST',
      });

      localStorage.removeItem('onboarding_session_id');
      onComplete(formData as OnboardingData);
    } catch (error) {
      console.error('Failed to complete onboarding:', error);
    }
  };

  const updateFormData = (section: keyof OnboardingData, data: any) => {
    setFormData(prev => ({ ...prev, [section]: data }));
  };

  const progressPercentage = (currentStep / 7) * 100;

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 py-8 px-4">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <div className="text-center mb-8">
          <h1 className="text-4xl font-bold text-gray-900 mb-2">Welcome to Your Wealth Journey</h1>
          <p className="text-gray-600">Let's get you started in just a few minutes</p>
        </div>

        {/* Progress Bar */}
        <div className="mb-8">
          <div className="flex justify-between mb-2">
            <span className="text-sm font-medium text-gray-700">Step {currentStep} of 7</span>
            <span className="text-sm font-medium text-gray-700">{Math.round(progressPercentage)}% Complete</span>
          </div>
          <div className="w-full bg-gray-200 rounded-full h-3">
            <div
              className="bg-gradient-to-r from-blue-500 to-indigo-600 h-3 rounded-full transition-all duration-500"
              style={{ width: `${progressPercentage}%` }}
            />
          </div>
        </div>

        {/* Step Indicators */}
        <div className="flex justify-between mb-8 overflow-x-auto">
          {STEPS.map((step) => {
            const Icon = step.icon;
            const isActive = step.number === currentStep;
            const isCompleted = step.number < currentStep;

            return (
              <div key={step.number} className="flex flex-col items-center min-w-[80px]">
                <div
                  className={`w-12 h-12 rounded-full flex items-center justify-center mb-2 transition-all ${
                    isCompleted
                      ? 'bg-green-500 text-white'
                      : isActive
                      ? 'bg-indigo-600 text-white ring-4 ring-indigo-200'
                      : 'bg-gray-200 text-gray-400'
                  }`}
                >
                  {isCompleted ? <Check className="w-6 h-6" /> : <Icon className="w-6 h-6" />}
                </div>
                <span className={`text-xs text-center ${isActive ? 'font-semibold' : 'text-gray-600'}`}>
                  {step.name}
                </span>
              </div>
            );
          })}
        </div>

        {/* Main Form Card */}
        <div className="bg-white rounded-2xl shadow-xl p-8">
          {/* Import step-specific components here */}
          {currentStep === 1 && (
            <PersonalInfoStep
              data={formData.personalInfo}
              onChange={(data) => updateFormData('personalInfo', data)}
              errors={errors}
            />
          )}
          {currentStep === 2 && (
            <EmploymentStep
              data={formData.employment}
              onChange={(data) => updateFormData('employment', data)}
              errors={errors}
            />
          )}
          {currentStep === 3 && (
            <GoalsStep
              data={formData.goals}
              onChange={(data) => updateFormData('goals', data)}
              errors={errors}
            />
          )}
          {currentStep === 4 && (
            <RiskAssessmentStep
              data={formData.riskAssessment}
              onChange={(data) => updateFormData('riskAssessment', data)}
              errors={errors}
            />
          )}
          {currentStep === 5 && (
            <DocumentUploadStep
              sessionId={sessionId!}
              documents={formData.documents || []}
              onChange={(data) => updateFormData('documents', data)}
              errors={errors}
            />
          )}
          {currentStep === 6 && (
            <AccountSetupStep
              data={formData.accountFunding}
              onChange={(data) => updateFormData('accountFunding', data)}
              errors={errors}
            />
          )}
          {currentStep === 7 && (
            <SignatureStep
              sessionId={sessionId!}
              signatures={formData.signatures || []}
              onChange={(data) => updateFormData('signatures', data)}
            />
          )}

          {/* Navigation Buttons */}
          <div className="flex justify-between mt-8 pt-6 border-t">
            <button
              onClick={prevStep}
              disabled={currentStep === 1}
              className="flex items-center px-6 py-3 text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              <ChevronLeft className="w-5 h-5 mr-2" />
              Previous
            </button>

            <div className="flex items-center gap-3">
              {isSaving && (
                <span className="text-sm text-gray-500 italic">Saving...</span>
              )}
              <button
                onClick={nextStep}
                className="flex items-center px-8 py-3 text-white bg-gradient-to-r from-blue-600 to-indigo-600 rounded-lg hover:from-blue-700 hover:to-indigo-700 transition-all shadow-lg hover:shadow-xl"
              >
                {currentStep === 7 ? 'Complete' : 'Continue'}
                {currentStep < 7 && <ChevronRight className="w-5 h-5 ml-2" />}
              </button>
            </div>
          </div>
        </div>

        {/* Auto-save Indicator */}
        <div className="text-center mt-4 text-sm text-gray-500">
          <Shield className="w-4 h-4 inline mr-2" />
          Your progress is automatically saved. You can resume anytime.
        </div>
      </div>
    </div>
  );
};

// Step Components (simplified - would be separate files)
const PersonalInfoStep: React.FC<any> = ({ data = {}, onChange, errors }) => (
  <div className="space-y-6">
    <h2 className="text-2xl font-bold text-gray-900 mb-6">Personal Information</h2>
    
    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">First Name *</label>
        <input
          type="text"
          value={data.firstName || ''}
          onChange={(e) => onChange({ ...data, firstName: e.target.value })}
          className={`w-full px-4 py-3 border rounded-lg focus:ring-2 focus:ring-indigo-500 ${
            errors.firstName ? 'border-red-500' : 'border-gray-300'
          }`}
        />
        {errors.firstName && <p className="text-red-500 text-sm mt-1">{errors.firstName}</p>}
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">Last Name *</label>
        <input
          type="text"
          value={data.lastName || ''}
          onChange={(e) => onChange({ ...data, lastName: e.target.value })}
          className={`w-full px-4 py-3 border rounded-lg focus:ring-2 focus:ring-indigo-500 ${
            errors.lastName ? 'border-red-500' : 'border-gray-300'
          }`}
        />
        {errors.lastName && <p className="text-red-500 text-sm mt-1">{errors.lastName}</p>}
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">Email *</label>
        <input
          type="email"
          value={data.email || ''}
          onChange={(e) => onChange({ ...data, email: e.target.value })}
          className={`w-full px-4 py-3 border rounded-lg focus:ring-2 focus:ring-indigo-500 ${
            errors.email ? 'border-red-500' : 'border-gray-300'
          }`}
        />
        {errors.email && <p className="text-red-500 text-sm mt-1">{errors.email}</p>}
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">Phone *</label>
        <input
          type="tel"
          value={data.phone || ''}
          onChange={(e) => onChange({ ...data, phone: e.target.value })}
          className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">Date of Birth *</label>
        <input
          type="date"
          value={data.dateOfBirth || ''}
          onChange={(e) => onChange({ ...data, dateOfBirth: e.target.value })}
          className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">SSN *</label>
        <input
          type="text"
          placeholder="XXX-XX-XXXX"
          value={data.ssn || ''}
          onChange={(e) => onChange({ ...data, ssn: e.target.value })}
          className={`w-full px-4 py-3 border rounded-lg focus:ring-2 focus:ring-indigo-500 ${
            errors.ssn ? 'border-red-500' : 'border-gray-300'
          }`}
          maxLength={11}
        />
        {errors.ssn && <p className="text-red-500 text-sm mt-1">{errors.ssn}</p>}
      </div>
    </div>

    <div>
      <label className="block text-sm font-medium text-gray-700 mb-2">Address *</label>
      <input
        type="text"
        placeholder="Street Address"
        value={data.addressLine1 || ''}
        onChange={(e) => onChange({ ...data, addressLine1: e.target.value })}
        className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 mb-3"
      />
      <input
        type="text"
        placeholder="Apt, Suite, etc. (optional)"
        value={data.addressLine2 || ''}
        onChange={(e) => onChange({ ...data, addressLine2: e.target.value })}
        className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
      />
    </div>

    <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">City *</label>
        <input
          type="text"
          value={data.city || ''}
          onChange={(e) => onChange({ ...data, city: e.target.value })}
          className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">State *</label>
        <select
          value={data.state || ''}
          onChange={(e) => onChange({ ...data, state: e.target.value })}
          className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
        >
          <option value="">Select State</option>
          <option value="CA">California</option>
          <option value="NY">New York</option>
          {/* Add all states */}
        </select>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">ZIP Code *</label>
        <input
          type="text"
          value={data.zipCode || ''}
          onChange={(e) => onChange({ ...data, zipCode: e.target.value })}
          className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
          maxLength={10}
        />
      </div>
    </div>
  </div>
);

// Other step components would follow similar patterns...
const EmploymentStep: React.FC<any> = ({ data = {}, onChange, errors }) => (
  <div className="space-y-6">
    <h2 className="text-2xl font-bold text-gray-900 mb-6">Employment & Income</h2>
    {/* Employment form fields */}
  </div>
);

const GoalsStep: React.FC<any> = ({ data = {}, onChange, errors }) => (
  <div className="space-y-6">
    <h2 className="text-2xl font-bold text-gray-900 mb-6">Financial Goals</h2>
    {/* Goals form fields */}
  </div>
);

const RiskAssessmentStep: React.FC<any> = ({ data = {}, onChange, errors }) => (
  <div className="space-y-6">
    <h2 className="text-2xl font-bold text-gray-900 mb-6">Risk Assessment</h2>
    {/* Risk questionnaire */}
  </div>
);

const DocumentUploadStep: React.FC<any> = ({ sessionId, documents, onChange, errors }) => (
  <div className="space-y-6">
    <h2 className="text-2xl font-bold text-gray-900 mb-6">Upload Documents</h2>
    {/* Document upload with OCR feedback */}
  </div>
);

const AccountSetupStep: React.FC<any> = ({ data = {}, onChange, errors }) => (
  <div className="space-y-6">
    <h2 className="text-2xl font-bold text-gray-900 mb-6">Account Setup</h2>
    {/* Account type selection */}
  </div>
);

const SignatureStep: React.FC<any> = ({ sessionId, signatures, onChange }) => (
  <div className="space-y-6">
    <h2 className="text-2xl font-bold text-gray-900 mb-6">Sign Agreements</h2>
    {/* E-signature integration */}
  </div>
);
