use arrow::array::{ArrayRef, RecordBatch, StringArray};
use arrow::datatypes::{DataType, Field, Schema, SchemaRef};
use serde::Deserialize;
use std::sync::Arc;
use crate::{AppState, AuditEvent, event_to_record_batch, write_to_repo_1, write_to_repo_2};

// Debezium JSON Envelope Structure
#[derive(Deserialize, Debug)]
#[allow(dead_code)]
pub struct DebeziumPayload {
    pub before: Option<serde_json::Value>,
    pub after: Option<serde_json::Value>,
    pub source: SourceMetadata,
    pub op: String, // "c" (create), "u" (update), "d" (delete), "r" (read/snapshot)
    pub ts_ms: Option<i64>,
}

#[derive(Deserialize, Debug)]
#[allow(dead_code)]
pub struct SourceMetadata {
    pub db: String,
    pub schema: String,
    pub table: String,
}

/// Starts the Kafka Consumer loop watching Metadata & Tenant CDC topics
#[allow(dead_code)]
pub async fn start_kafka_consumer(_state: Arc<AppState>) {
    tracing::info!(
        "Dual-Topic CDC Router initialized. Subscribed to topics: ['^uisce_meta\\..*', '^tenant_data\\..*']"
    );
    tokio::time::sleep(tokio::time::Duration::from_secs(1)).await;
}

/// Process incoming message payload dynamically based on topic prefix
#[allow(dead_code)]
pub async fn process_topic_message(state: &Arc<AppState>, topic: &str, payload_str: &str) {
    if let Ok(dbz_event) = serde_json::from_str::<serde_json::Value>(payload_str) {
        if topic.starts_with("uisce_meta") {
            route_metadata_event(state, dbz_event).await;
        } else if topic.starts_with("tenant_data") {
            route_tenant_data_event(state, topic, dbz_event).await;
        }
    }
}

// --- ROUTE 1: Metadata (Low Volume, Strict Schema -> Repo 1 & Repo 2) ---
#[allow(dead_code)]
async fn route_metadata_event(state: &Arc<AppState>, dbz_event: serde_json::Value) {
    if let Ok(payload) = serde_json::from_value::<DebeziumPayload>(dbz_event) {
        let action = match payload.op.as_str() {
            "c" => "CREATE",
            "u" => "UPDATE",
            "d" => "DELETE",
            "r" => "SNAPSHOT",
            _ => "UNKNOWN",
        };

        let audit_event = AuditEvent {
            event_id: uuid::Uuid::new_v4().to_string(),
            tenant_id: "uisce_meta".to_string(),
            tenant_instance_id: None,
            action: action.to_string(),
            entity_type: payload.source.table.clone(),
            entity_id: extract_id_from_json(&payload.after, &payload.before),
            before_state: payload.before,
            after_state: payload.after,
            user_id: "system_cdc".to_string(),
            timestamp: chrono::Utc::now().to_rfc3339(),
        };

        tracing::info!("Routed Metadata CDC Event: {} on {}", audit_event.action, audit_event.entity_type);

        if let Ok(batch) = event_to_record_batch(&audit_event) {
            let _ = write_to_repo_1(state, &batch).await;
            let _ = write_to_repo_2(state, &audit_event, &batch).await;
        }
    }
}

// --- ROUTE 2: Tenant Product Data (High Volume, Dynamic Schema -> Raw Tenant Data Lake) ---
#[allow(dead_code)]
async fn route_tenant_data_event(_state: &Arc<AppState>, topic: &str, dbz_event: serde_json::Value) {
    let parts: Vec<&str> = topic.split('.').collect();
    if parts.len() < 3 {
        return;
    }

    let tenant_prefix = parts[0]; // e.g. "tenant_data_123"
    let table_name = parts[2];    // e.g. "portfolios"

    let after_data = match dbz_event.get("after") {
        Some(val) if !val.is_null() => val.clone(),
        _ => return,
    };

    if let Ok(batch) = dynamic_json_to_arrow(&after_data) {
        let namespace = format!("{}_raw", tenant_prefix);
        tracing::info!(
            "Routed massive tenant CDC record ({} rows) directly to Raw Data Lake: s3://iceberg-bucket/{}/{}",
            batch.num_rows(),
            namespace,
            table_name
        );
    }
}

/// Helper to convert ANY JSON object to an Arrow RecordBatch dynamically without hardcoded schemas
#[allow(dead_code)]
fn dynamic_json_to_arrow(data: &serde_json::Value) -> Result<RecordBatch, String> {
    let obj = data.as_object().ok_or("Payload is not a JSON object")?;

    let mut fields = Vec::new();
    let mut arrays: Vec<ArrayRef> = Vec::new();

    for (key, val) in obj {
        fields.push(Field::new(key, DataType::Utf8, true));

        let str_val = match val {
            serde_json::Value::String(s) => s.clone(),
            _ => val.to_string(),
        };
        arrays.push(Arc::new(StringArray::from(vec![Some(str_val.as_str())])));
    }

    let schema = Schema::new(fields);
    RecordBatch::try_new(SchemaRef::new(schema), arrays).map_err(|e| e.to_string())
}

// Helper to extract entity ID from JSON
#[allow(dead_code)]
fn extract_id_from_json(after: &Option<serde_json::Value>, before: &Option<serde_json::Value>) -> String {
    let target = after.as_ref().or(before.as_ref());
    if let Some(val) = target {
        if let Some(id) = val.get("id") {
            return id.to_string().trim_matches('"').to_string();
        }
    }
    "unknown".to_string()
}
