import React, { useState, useCallback, useEffect, useMemo, useRef } from 'react';
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { CheckCircle2, Clock, Loader2, AlertCircle } from 'lucide-react';
import { useTenant } from '../contexts/TenantContext';

interface EmployeeData {
  firstName: string;
  lastName: string;
  email: string;
  department: string;
  jobTitle: string;
  managerID: string;
  startDate: string;
  salary: number;
}
interface BPInstanceStatus {
  instance_id: string;
  process_id: string;
  process_name: string;
  entity_id: string;
  entity_type: string;
  current_step: number;
  status: string;
  started_at: string;
  current_step_started_at: string;
  current_step_due_at: string;
}

interface EventEntry {
  id: string;
  message: string;
  timestamp: string;
}

// Email and department are handled in the component state
const timelineSteps = [
  { order: 1, name: 'Employee Intake', icon: '👤', hint: 'Collect profile + offer details' },
  { order: 2, name: 'Background Check', icon: '🛡️', hint: 'Validate docs & compliance' },
  { order: 3, name: 'Manager Approval', icon: '✅', hint: 'Manager + HR approval' },
  { order: 4, name: 'Provision Systems', icon: '⚙️', hint: 'Provision payroll + access' },
];

const hireEmployeeTemplate = {
  name: 'HireEmployee',
  description: 'End-to-end hiring workflow with validation, approvals, and provisioning.',
  steps: [
    {
      id: 'collect-info',
      type: 'data_entry',
      title: 'Collect Employee Info',
      duration_hours: 4,
      assignee_role: 'Recruiter',
      fields: ['first_name', 'last_name', 'email', 'department'],
    },
    {
      id: 'background-check',
      type: 'validate',
      title: 'Background Check',
      duration_hours: 24,
      rules: ['criminal_check', 'employment_history'],
    },
    {
      id: 'manager-approval',
      type: 'approve',
      title: 'Manager Approval',
      duration_hours: 48,
      assignee_role: 'Hiring Manager',
    },
    {
      id: 'provisioning',
      type: 'notify',
      title: 'System Provisioning',
      duration_hours: 12,
      action: 'notify_workday',
    },
  ],
  transitions: [
    { from: 'collect-info', to: 'background-check' },
    { from: 'background-check', to: 'manager-approval' },
    { from: 'manager-approval', to: 'provisioning' },
  ],
  audit: [
    { key: 'owner', value: 'Talent Operations' },
    { key: 'sla_hours', value: '72' },
  ],
};

export const HireEmployeeDemo: React.FC = () => {
  const { tenant, datasource } = useTenant();
  const scopeMissing = !tenant?.id || !datasource?.id;
  const [employee, setEmployee] = useState<EmployeeData>({
    firstName: '',
    lastName: '',
    email: '',
    department: 'Engineering',
    jobTitle: '',
    managerID: '',
    startDate: new Date().toISOString().split('T')[0],
    salary: 0,
  });
  const [bpId, setBpId] = useState<string | null>(null);
  const [instanceId, setInstanceId] = useState<string | null>(null);
  const [status, setStatus] = useState<BPInstanceStatus | null>(null);
  const [events, setEvents] = useState<EventEntry[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState({ create: false, start: false, approve: false, polling: false });
  const stepTracker = useRef<{ step: number | null; state: string | null }>({ step: null, state: null });

  const scopedFetch = useCallback(
    async (path: string, init?: RequestInit) => {
      if (scopeMissing) {
        throw new Error('Select a tenant + datasource to run the demo.');
      }
      const response = await fetch(path, {
        ...init,
        headers: {
          'Content-Type': 'application/json',
          ...(init?.headers || {}),
        },
      });
      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || response.statusText);
      }
      if (response.status === 204) {
        return null;
      }
      return response.json();
    },
    [scopeMissing]
  );

  const appendEvent = useCallback((message: string) => {
    setEvents((prev) => [
      { id: `${Date.now()}-${Math.random()}`, message, timestamp: new Date().toISOString() },
      ...prev.slice(0, 24),
    ]);
  }, []);

  const handleCreateBP = async () => {
    try {
      setBusy((prev) => ({ ...prev, create: true }));
      setError(null);
      const data = (await scopedFetch('/api/bp', {
        method: 'POST',
        body: JSON.stringify(hireEmployeeTemplate),
      })) as { id: string };
      setBpId(data.id);
      setInstanceId(null);
      setStatus(null);
      setEvents([]);
      appendEvent(`Created HireEmployee process (${data.id}).`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create business process');
    } finally {
      setBusy((prev) => ({ ...prev, create: false }));
    }
  };

  const handleStartExecution = async () => {
    if (!bpId) return;
    try {
      setBusy((prev) => ({ ...prev, start: true }));
      setError(null);
      const payload = {
        entity_id: `emp-${Date.now()}`,
        entity_type: 'employee',
        data: {
          first_name: employee.firstName,
          last_name: employee.lastName,
          email: employee.email,
          department: employee.department,
          job_title: employee.jobTitle,
          manager_id: employee.managerID,
          start_date: employee.startDate,
          salary: employee.salary,
        },
      };
      const data = (await scopedFetch(`/api/bp/${bpId}/start`, {
        method: 'POST',
        body: JSON.stringify(payload),
      })) as { instance_id: string };
      setInstanceId(data.instance_id);
      setStatus(null);
      appendEvent(`Started execution for ${data.instance_id}.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start execution');
    } finally {
      setBusy((prev) => ({ ...prev, start: false }));
    }
  };

  const fetchStatus = useCallback(
    async (silent = false) => {
      if (!instanceId) return;
      try {
        if (!silent) {
          setBusy((prev) => ({ ...prev, polling: true }));
        }
        const data = (await scopedFetch(`/api/bp/instance/${instanceId}`)) as BPInstanceStatus;
        setStatus(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch status');
      } finally {
        if (!silent) {
          setBusy((prev) => ({ ...prev, polling: false }));
        }
      }
    },
    [instanceId, scopedFetch]
  );

  useEffect(() => {
    if (!instanceId) return;
    const interval = setInterval(() => {
      fetchStatus(true);
    }, 4000);
    return () => clearInterval(interval);
  }, [instanceId, fetchStatus]);

  useEffect(() => {
    if (!status) return;
    const tracker = stepTracker.current;
    if (tracker.step !== status.current_step) {
      appendEvent(`Moved to Step ${status.current_step}`);
      tracker.step = status.current_step;
    }
    if (tracker.state !== status.status) {
      appendEvent(`Status changed to ${status.status}`);
      tracker.state = status.status;
    }
  }, [status, appendEvent]);

  const handleApprove = async () => {
    if (!instanceId) return;
    try {
      setBusy((prev) => ({ ...prev, approve: true }));
      setError(null);
      await scopedFetch(`/api/bp/instance/${instanceId}/approve`, {
        method: 'POST',
        body: JSON.stringify({
          decision: 'approved',
          comment: 'Auto-approved via HireEmployee demo',
          reason: 'demo',
        }),
      });
      appendEvent('Approval submitted for current step.');
      await fetchStatus();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to approve step');
    } finally {
      setBusy((prev) => ({ ...prev, approve: false }));
    }
  };

  const getStepStatus = (order: number) => {
    if (!status) return 'pending';
    if (status.status === 'completed' && order <= timelineSteps.length) return 'completed';
    if (status.current_step > order) return 'completed';
    if (status.current_step === order && status.status !== 'completed') return 'active';
    return 'pending';
  };

  const steps = timelineSteps;
  const isComplete = status?.status === 'completed';

  const handleInputChange = (field: keyof EmployeeData, value: string | number) => {
    setEmployee((prev) => ({ ...prev, [field]: value }));
  };

  if (scopeMissing) {
    return (
      <div className="max-w-3xl mx-auto p-10">
        <Card>
          <CardContent className="py-10 text-center space-y-4">
            <div className="text-4xl">🔐</div>
            <CardTitle>Select a tenant + datasource</CardTitle>
            <p className="text-gray-600">
              The HireEmployee demo calls real `/api/bp` endpoints. Use the Fabric Builder scope picker to choose a
              tenant and datasource (or seed `localStorage` with `selected_tenant` / `selected_datasource`) before running the demo.
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="max-w-5xl mx-auto p-8 space-y-8">
      <div className="flex justify-between items-start">
        <div>
          <h1 className="text-3xl font-bold">Hire Employee - Live Demo</h1>
          <p className="text-gray-600 mt-1">
            Experience Workday-level business process automation
          </p>
        </div>
        {instanceId && (
          <Badge variant="outline" className="text-sm">
            Instance: {instanceId.slice(0, 8)}...
          </Badge>
        )}
      </div>

      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center justify-between">
            {steps.map((s, idx) => (
              <React.Fragment key={s.order}>
                <div className="flex flex-col items-center">
                  <div
                    className={`
                      w-12 h-12 rounded-full flex items-center justify-center text-xl
                      ${getStepStatus(s.order) === 'completed' ? 'bg-green-500 text-white' :
                        getStepStatus(s.order) === 'active' ? 'bg-blue-500 text-white animate-pulse' :
                        'bg-gray-200 text-gray-500'}
                    `}
                  >
                    {getStepStatus(s.order) === 'completed' ? <CheckCircle2 size={24} /> :
                     getStepStatus(s.order) === 'active' ? <Loader2 size={24} className="animate-spin" /> :
                     s.icon}
                  </div>
                  <span className="text-sm mt-2 text-center">{s.name}</span>
                  <p className="text-xs text-gray-500 mt-1">{s.hint}</p>
                </div>
                {idx < steps.length - 1 && (
                  <div className={`flex-1 h-1 mx-4 ${
                    getStepStatus(s.order) === 'completed' ? 'bg-green-500' : 'bg-gray-200'
                  }`} />
                )}
              </React.Fragment>
            ))}
          </div>
        </CardContent>
      </Card>

      {error && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}
      {!bpId && (
        <Card>
          <CardHeader>
            <CardTitle>Create HireEmployee Business Process</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <p className="text-gray-600">
              This button seeds the `HireEmployee` definition via <code className="bg-gray-100 px-1 rounded text-xs">POST /api/bp</code> and
              stores it in the multi-tenant catalog.
            </p>
            <Button onClick={handleCreateBP} disabled={busy.create} className="w-full">
              {busy.create ? (
                <>
                  <Loader2 className="mr-2 animate-spin" size={16} />
                  Creating process...
                </>
              ) : (
                'Create Sample Process'
              )}
            </Button>
          </CardContent>
        </Card>
      )}

      {bpId && (
        <Card>
          <CardHeader>
            <CardTitle>Employee Details</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="firstName">First Name</Label>
                <Input
                  id="firstName"
                  value={employee.firstName}
                  onChange={(e) => handleInputChange('firstName', e.target.value)}
                  placeholder="John"
                />
              </div>
              <div>
                <Label htmlFor="lastName">Last Name</Label>
                <Input
                  id="lastName"
                  value={employee.lastName}
                  onChange={(e) => handleInputChange('lastName', e.target.value)}
                  placeholder="Doe"
                />
              </div>
            </div>

            <div>
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                value={employee.email}
                onChange={(e) => handleInputChange('email', e.target.value)}
                placeholder="john.doe@company.com"
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="department">Department</Label>
                <Select
                  value={employee.department}
                  onValueChange={(value) => handleInputChange('department', value)}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="Engineering">Engineering</SelectItem>
                    <SelectItem value="Product">Product</SelectItem>
                    <SelectItem value="Sales">Sales</SelectItem>
                    <SelectItem value="Marketing">Marketing</SelectItem>
                    <SelectItem value="Finance">Finance</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div>
                <Label htmlFor="jobTitle">Job Title</Label>
                <Input
                  id="jobTitle"
                  value={employee.jobTitle}
                  onChange={(e) => handleInputChange('jobTitle', e.target.value)}
                  placeholder="Senior Software Engineer"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="startDate">Start Date</Label>
                <Input
                  id="startDate"
                  type="date"
                  value={employee.startDate}
                  onChange={(e) => handleInputChange('startDate', e.target.value)}
                />
              </div>
              <div>
                <Label htmlFor="salary">Annual Salary ($)</Label>
                <Input
                  id="salary"
                  type="number"
                  value={employee.salary || ''}
                  onChange={(e) => handleInputChange('salary', parseFloat(e.target.value))}
                  placeholder="120000"
                />
              </div>
            </div>

            <Button
              onClick={handleStartExecution}
              disabled={busy.start || !employee.firstName || !employee.email}
              className="w-full"
            >
              {busy.start ? (
                <>
                  <Loader2 className="mr-2 animate-spin" size={16} />
                  Starting workflow...
                </>
              ) : (
                'Start Hiring Process'
              )}
            </Button>
          </CardContent>
        </Card>
      )}

      {instanceId && status && (
        <Card>
          <CardHeader>
            <CardTitle>Workflow Status</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <Alert>
              <AlertDescription>
                <div className="flex items-center justify-between">
                  <span>Status: <strong>{status.status}</strong></span>
                  <Badge>{status.process_name}</Badge>
                </div>
              </AlertDescription>
            </Alert>

            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <p className="text-gray-500">Current Step</p>
                <p className="font-semibold">{status.current_step}</p>
              </div>
              <div>
                <p className="text-gray-500">Entity</p>
                <p className="font-semibold">{status.entity_id} ({status.entity_type})</p>
              </div>
              <div>
                <p className="text-gray-500">Started</p>
                <p className="font-semibold">{new Date(status.started_at).toLocaleString()}</p>
              </div>
              <div>
                <p className="text-gray-500">Step Due</p>
                <p className="font-semibold">{new Date(status.current_step_due_at).toLocaleString()}</p>
              </div>
            </div>

            <div className="flex gap-3">
              <Button onClick={() => fetchStatus()} disabled={busy.polling} variant="secondary">
                {busy.polling ? <Loader2 className="mr-2 animate-spin" size={16} /> : <Clock className="mr-2" size={16} />}
                Poll Status
              </Button>
              <Button onClick={handleApprove} disabled={busy.approve || isComplete} variant="outline">
                {busy.approve ? <Loader2 className="mr-2 animate-spin" size={16} /> : null}
                Approve Current Step
              </Button>
            </div>

            {isComplete && (
              <div>
                <Alert className="bg-green-50 border-green-200">
                  <AlertDescription className="text-green-800">
                    🎉 Employee successfully hired! All systems provisioned and ready for {employee.startDate}.
                  </AlertDescription>
                </Alert>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {events.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Event Log</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 max-h-64 overflow-y-auto text-sm">
              {events.map((event) => (
                <div key={event.id} className="flex items-center gap-3 p-2 rounded border border-gray-100">
                  <Clock size={14} className="text-gray-400" />
                  <div>
                    <p className="font-semibold text-gray-800">{event.message}</p>
                    <p className="text-xs text-gray-500">{new Date(event.timestamp).toLocaleTimeString()}</p>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
};
