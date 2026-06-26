export interface AutomationPolicy {
  id: string;
  policy_id: string;
  description: string;
  trigger: string;
  action: string;
  is_enabled: boolean;
}

export interface AutomationLog {
  id: string;
  timestamp: string;
  action: string;
  target_type: string;
  target_id: string;
  details: Record<string, any>;
  status: string;
}