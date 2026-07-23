import { AutomationPolicy, AutomationLog } from './types';

// This is a mock API client. In a real application, this would use
// fetch or axios to make network requests to your backend.

const API_BASE = '/api'; // Adjust if your API is hosted elsewhere

export async function listAutomationPolicies(): Promise<AutomationPolicy[]> {
  const response = await fetch(`${API_BASE}/automation/policies`);
  if (!response.ok) throw new Error('Failed to fetch automation policies');
  return response.json();
}

export async function listAutomationLogs(): Promise<AutomationLog[]> {
  const response = await fetch(`${API_BASE}/automation/logs`);
  if (!response.ok) throw new Error('Failed to fetch automation logs');
  return response.json();
}

export async function runAutomationCycle(): Promise<void> {
  const response = await fetch(`${API_BASE}/automation/run`, { method: 'POST' });
  if (!response.ok) throw new Error('Failed to run automation cycle');
}