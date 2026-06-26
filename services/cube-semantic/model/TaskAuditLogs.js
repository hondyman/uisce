console.log('=== EXECUTING SCHEMA FILE: TaskAuditLogs.js ===');
cube(`task_audit_logs`, {
  sql: `SELECT * FROM iceberg.default.audit_logs`,
  dataSource: `trino`,
  
  measures: {
    count: {
      type: `count`
    }
  },

  dimensions: {
    id: {
      sql: `id`,
      type: `string`,
      primaryKey: true
    },
    
    event_type: {
      sql: `event_type`,
      type: `string`
    },

    created_at: {
      sql: `created_at`,
      type: `time`
    },
    
    tenant_id: {
      sql: `tenant_id`,
      type: `string`
    },
    
    datasource_id: {
      sql: `datasource_id`,
      type: `string`
    },
    
    action: {
      sql: `action`,
      type: `string`
    },
    
    resource: {
      sql: `resource`,
      type: `string`
    },
    
    details: {
      sql: `details`,
      type: `string`
    }
  }
});
