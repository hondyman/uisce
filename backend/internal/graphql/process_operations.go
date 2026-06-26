package graphql

// Business Process GraphQL Operations

// ============================================================================
// Process Definition Queries
// ============================================================================

// GetBusinessProcessByKey fetches a process definition
const GetBusinessProcessByKey = `
query GetBusinessProcessByKey($key: String!) {
  business_processes(where: { key: { _eq: $key } }, limit: 1) {
    id
    tenant_id
    key
    name
    display_name
    description
    category
    status
    version
    is_system
    created_at
  }
}
`

// ListBusinessProcesses fetches all active processes
const ListBusinessProcesses = `
query ListBusinessProcesses($category: String) {
  business_processes(
    where: { status: { _eq: "active" }, category: { _eq: $category } }
    order_by: { name: asc }
  ) {
    id
    key
    name
    display_name
    description
    category
    is_system
  }
}
`

// GetProcessWithSteps fetches a process with all its steps
const GetProcessWithSteps = `
query GetProcessWithSteps($id: String!) {
  business_processes_by_pk(id: $id) {
    id
    key
    name
    display_name
    description
    category
    status
    process_steps(order_by: { sequence: asc }) {
      id
      key
      name
      display_name
      step_type
      sequence
      config
      is_required
    }
  }
}
`

// ============================================================================
// Process Step Queries
// ============================================================================

// GetProcessSteps fetches steps for a process
const GetProcessSteps = `
query GetProcessSteps($process_id: String!) {
  process_steps(
    where: { process_id: { _eq: $process_id } }
    order_by: { sequence: asc }
  ) {
    id
    tenant_id
    process_id
    key
    name
    display_name
    step_type
    sequence
    config
    is_required
  }
}
`

// ============================================================================
// Process Instance Queries
// ============================================================================

// GetProcessInstance fetches a process instance
const GetProcessInstance = `
query GetProcessInstance($id: String!) {
  process_instances_by_pk(id: $id) {
    id
    tenant_id
    process_id
    entity_type
    entity_id
    current_step_id
    status
    started_at
    completed_at
    data
    created_by
    process {
      id
      key
      name
      display_name
    }
    current_step {
      id
      key
      name
      step_type
    }
  }
}
`

// GetProcessInstanceWithHistory fetches instance with full history
const GetProcessInstanceWithHistory = `
query GetProcessInstanceWithHistory($id: String!) {
  process_instances_by_pk(id: $id) {
    id
    process_id
    entity_type
    entity_id
    current_step_id
    status
    started_at
    completed_at
    data
    created_by
    step_histories(order_by: { created_at: asc }) {
      id
      step_id
      action
      actor
      comments
      data
      created_at
    }
  }
}
`

// ListInstancesForEntity fetches all instances for an entity
const ListInstancesForEntity = `
query ListInstancesForEntity($entity_type: String!, $entity_id: String!) {
  process_instances(
    where: { entity_type: { _eq: $entity_type }, entity_id: { _eq: $entity_id } }
    order_by: { started_at: desc }
  ) {
    id
    process_id
    status
    started_at
    completed_at
    created_by
    process {
      key
      name
      display_name
    }
  }
}
`

// ListPendingApprovals fetches instances waiting for a user's approval
const ListPendingApprovals = `
query ListPendingApprovals($actor: String!) {
  process_instances(
    where: { status: { _eq: "in_progress" } }
    order_by: { started_at: desc }
  ) {
    id
    process_id
    entity_type
    entity_id
    current_step_id
    started_at
    current_step {
      name
      step_type
      config
    }
    process {
      key
      name
      display_name
    }
  }
}
`

// ============================================================================
// Process Instance Mutations
// ============================================================================

// InsertProcessInstance creates a new process instance
const InsertProcessInstance = `
mutation InsertProcessInstance($object: process_instances_insert_input!) {
  insert_process_instances_one(object: $object) {
    id
    status
    started_at
  }
}
`

// UpdateProcessInstanceStep updates current step
const UpdateProcessInstanceStep = `
mutation UpdateProcessInstanceStep($id: String!, $step_id: String!, $status: String!) {
  update_process_instances_by_pk(
    pk_columns: { id: $id }
    _set: { current_step_id: $step_id, status: $status }
  ) {
    id
    current_step_id
    status
  }
}
`

// UpdateProcessInstanceStatus updates status
const UpdateProcessInstanceStatus = `
mutation UpdateProcessInstanceStatus($id: String!, $status: String!) {
  update_process_instances_by_pk(
    pk_columns: { id: $id }
    _set: { status: $status }
  ) {
    id
    status
  }
}
`

// CompleteProcessInstance marks as completed
const CompleteProcessInstance = `
mutation CompleteProcessInstance($id: String!, $completed_at: timestamptz!) {
  update_process_instances_by_pk(
    pk_columns: { id: $id }
    _set: { status: "completed", completed_at: $completed_at }
  ) {
    id
    status
    completed_at
  }
}
`

// UpdateProcessInstanceData updates instance data
const UpdateProcessInstanceData = `
mutation UpdateProcessInstanceData($id: String!, $data: jsonb!) {
  update_process_instances_by_pk(
    pk_columns: { id: $id }
    _set: { data: $data }
  ) {
    id
  }
}
`

// ============================================================================
// Step History Mutations
// ============================================================================

// InsertStepHistory records a step action
const InsertStepHistory = `
mutation InsertStepHistory($object: step_history_insert_input!) {
  insert_step_history_one(object: $object) {
    id
    action
    created_at
  }
}
`

// GetStepHistory fetches history for an instance
const GetStepHistory = `
query GetStepHistory($instance_id: String!) {
  step_history(
    where: { instance_id: { _eq: $instance_id } }
    order_by: { created_at: asc }
  ) {
    id
    instance_id
    step_id
    action
    actor
    comments
    data
    created_at
    step {
      name
      display_name
      step_type
    }
  }
}
`

// ============================================================================
// Dashboard Queries
// ============================================================================

// GetProcessDashboard fetches process metrics
const GetProcessDashboard = `
query GetProcessDashboard {
  total: process_instances_aggregate { aggregate { count } }
  in_progress: process_instances_aggregate(where: { status: { _eq: "in_progress" } }) { aggregate { count } }
  completed: process_instances_aggregate(where: { status: { _eq: "completed" } }) { aggregate { count } }
  rejected: process_instances_aggregate(where: { status: { _eq: "rejected" } }) { aggregate { count } }
  
  by_process: business_processes {
    id
    key
    name
    instances_aggregate {
      aggregate { count }
    }
  }
}
`
