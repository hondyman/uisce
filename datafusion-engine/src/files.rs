//! File endpoints for the uisce data pipeline.
//!
//! - POST /files/profile  : infer schema + sample rows (analyst preview)
//! - POST /files/read     : stream a file as NDJSON, in file order
//! - POST /files/convert  : NDJSON spool -> csv/json/parquet (export)
//! - POST /files/write    : NDJSON request body -> csv/json/parquet (export)
//!
//! All URIs resolve under DATAFUSION_FILE_ROOT (default /data/files); absolute
//! paths, `..` and symlinks that escape the root are rejected. Tenant scoping
//! is the caller's job: the Go side prefixes every path with the tenant id.
//!
//! CSV is read as all-strings on purpose: the pipeline's map/validate nodes
//! parse and report bad values per row (with plain-language reasons) instead
//! of DataFusion failing the whole scan on the first bad cell.

use axum::{body::Body, http::StatusCode, response::Response, Json};
use bytes::Bytes;
use datafusion::arrow::array::RecordBatch;
use datafusion::arrow::datatypes::{DataType, SchemaRef};
use datafusion::arrow::json::LineDelimitedWriter;
use datafusion::parquet::arrow::ArrowWriter;
use datafusion::prelude::*;
use futures::StreamExt;
use serde::{Deserialize, Serialize};
use std::fs::File;
use std::path::{Component, Path, PathBuf};

type ApiErr = (StatusCode, String);

fn bad(msg: impl ToString) -> ApiErr {
    (StatusCode::BAD_REQUEST, msg.to_string())
}
fn internal(msg: impl ToString) -> ApiErr {
    (StatusCode::INTERNAL_SERVER_ERROR, msg.to_string())
}

fn root() -> PathBuf {
    PathBuf::from(std::env::var("DATAFUSION_FILE_ROOT").unwrap_or_else(|_| "/data/files".into()))
}

/// Resolve a file:// (or bare relative) URI under the file root.
pub fn resolve(uri: &str) -> Result<PathBuf, ApiErr> {
    let rel = uri.strip_prefix("file://").unwrap_or(uri);
    if uri.contains("://") && !uri.starts_with("file://") {
        return Err(bad("only file:// URIs are supported by the engine; stage remote files first"));
    }
    let rel = rel.trim_start_matches('/');
    let p = Path::new(rel);
    if rel.is_empty() {
        return Err(bad("empty path"));
    }
    for c in p.components() {
        if !matches!(c, Component::Normal(_)) {
            return Err(bad("path must be relative and must not contain '..'"));
        }
    }
    let root = root();
    let full = root.join(p);
    // Symlink escape check on the deepest existing ancestor.
    let mut probe = full.clone();
    while !probe.exists() {
        if !probe.pop() {
            break;
        }
    }
    if let (Ok(cr), Ok(cp)) = (root.canonicalize(), probe.canonicalize()) {
        if !cp.starts_with(&cr) {
            return Err(bad("path escapes the file root"));
        }
    }
    Ok(full)
}

#[derive(Deserialize, Clone)]
pub struct FileSpec {
    pub uri: String,
    pub format: String, // csv | json | parquet
    #[serde(default)]
    pub delimiter: Option<String>,
    #[serde(default = "yes")]
    pub has_header: bool,
    /// Column names by position; required for headerless CSV.
    #[serde(default)]
    pub columns: Vec<String>,
}
fn yes() -> bool {
    true
}

fn delim(s: &Option<String>) -> Result<u8, ApiErr> {
    match s.as_deref() {
        None | Some("") => Ok(b','),
        Some("\\t") | Some("tab") => Ok(b'\t'),
        Some(d) if d.len() == 1 => Ok(d.as_bytes()[0]),
        Some(_) => Err(bad("delimiter must be a single character")),
    }
}

fn ctx() -> SessionContext {
    // One partition + no scan repartitioning => rows come back in file order,
    // which the pipeline relies on for _source_row_num.
    let cfg = SessionConfig::new()
        .with_target_partitions(1)
        .with_repartition_file_scans(false);
    SessionContext::new_with_config(cfg)
}

async fn open(spec: &FileSpec, typed: bool) -> Result<DataFrame, ApiErr> {
    let path = resolve(&spec.uri)?;
    let ps = path.to_str().ok_or_else(|| bad("non-utf8 path"))?.to_string();
    if !path.exists() {
        return Err((StatusCode::NOT_FOUND, format!("file not found: {}", spec.uri)));
    }
    let c = ctx();
    // DataFusion only reads paths with the format's default extension; the
    // format is declared, so accept the file's own (.txt, .psv, .ndjson, ...).
    let ext_owned = path
        .extension()
        .and_then(|e| e.to_str())
        .map(|e| format!(".{e}"))
        .unwrap_or_default();
    let ext = ext_owned.as_str();
    match spec.format.as_str() {
        "csv" => {
            let d = delim(&spec.delimiter)?;
            let mut opts = CsvReadOptions::new().has_header(spec.has_header).delimiter(d).file_extension(ext);
            let schema;
            if !typed {
                if spec.has_header {
                    // zero inference records => every column is Utf8, names from header
                    opts = opts.schema_infer_max_records(0);
                } else {
                    if spec.columns.is_empty() {
                        return Err(bad("headerless csv needs column names"));
                    }
                    schema = datafusion::arrow::datatypes::Schema::new(
                        spec.columns
                            .iter()
                            .map(|n| datafusion::arrow::datatypes::Field::new(n, DataType::Utf8, true))
                            .collect::<Vec<_>>(),
                    );
                    return c.read_csv(&ps, opts.schema(&schema)).await.map_err(internal);
                }
            } else {
                opts = opts.schema_infer_max_records(1000);
            }
            c.read_csv(&ps, opts).await.map_err(|e| bad(format!("cannot read csv: {e}")))
        }
        "json" => {
            let mut jo = NdJsonReadOptions::default().file_extension(ext);
            jo.schema_infer_max_records = 100_000;
            c.read_json(&ps, jo).await
                .map_err(|e| bad(format!("cannot read json (expected one object per line): {e}")))
        }
        "parquet" => c
            .read_parquet(&ps, ParquetReadOptions { file_extension: ext, ..Default::default() })
            .await
            .map_err(|e| bad(format!("cannot read parquet: {e}"))),
        f => Err(bad(format!("unsupported format {f}"))),
    }
}

fn type_name(t: &DataType) -> &'static str {
    use DataType::*;
    match t {
        Int8 | Int16 | Int32 | Int64 | UInt8 | UInt16 | UInt32 | UInt64 => "int",
        Float16 | Float32 | Float64 => "float",
        Decimal128(_, _) | Decimal256(_, _) => "decimal",
        Boolean => "bool",
        Date32 | Date64 => "date",
        Timestamp(_, _) => "timestamp",
        _ => "string",
    }
}

#[derive(Serialize)]
pub struct ColumnInfo {
    name: String,
    #[serde(rename = "type")]
    ty: &'static str,
    nullable: bool,
}

#[derive(Deserialize)]
pub struct ProfileReq {
    #[serde(flatten)]
    file: FileSpec,
    #[serde(default = "default_sample")]
    sample_rows: usize,
    #[serde(default)]
    count_rows: bool,
}
fn default_sample() -> usize {
    20
}

#[derive(Serialize)]
pub struct ProfileResp {
    columns: Vec<ColumnInfo>,
    sample: Vec<serde_json::Map<String, serde_json::Value>>,
    row_count: Option<usize>,
}

pub async fn profile(Json(req): Json<ProfileReq>) -> Result<Json<ProfileResp>, ApiErr> {
    // Typed open: the analyst sees inferred types (int/date/...) in the preview.
    let df = open(&req.file, true).await?;
    let columns = df
        .schema()
        .fields()
        .iter()
        .map(|f| ColumnInfo {
            name: f.name().clone(),
            ty: type_name(f.data_type()),
            nullable: f.is_nullable(),
        })
        .collect();

    let n = req.sample_rows.min(200);
    let batches = df.clone().limit(0, Some(n)).map_err(internal)?.collect().await.map_err(internal)?;
    let refs: Vec<&RecordBatch> = batches.iter().collect();
    #[allow(deprecated)]
    let sample = datafusion::arrow::json::writer::record_batches_to_json_rows(&refs).map_err(internal)?;

    let row_count = if req.count_rows {
        Some(df.count().await.map_err(internal)?)
    } else {
        None
    };
    Ok(Json(ProfileResp { columns, sample, row_count }))
}

/// Stream the file as NDJSON. CSV values are strings; json/parquet keep native types.
pub async fn read(Json(spec): Json<FileSpec>) -> Result<Response, ApiErr> {
    let df = open(&spec, false).await?;
    let stream = df.execute_stream().await.map_err(internal)?;
    let body = Body::from_stream(stream.map(|r| -> Result<Bytes, std::io::Error> {
        let batch = r.map_err(|e| std::io::Error::new(std::io::ErrorKind::Other, e.to_string()))?;
        let mut buf = Vec::new();
        {
            // Explicit nulls: every column is present on every line, so the
            // caller can tell an empty cell from a missing column.
            let mut w = datafusion::arrow::json::WriterBuilder::new()
                .with_explicit_nulls(true)
                .build::<_, datafusion::arrow::json::writer::LineDelimited>(&mut buf);
            w.write(&batch).map_err(|e| std::io::Error::new(std::io::ErrorKind::Other, e.to_string()))?;
            w.finish().map_err(|e| std::io::Error::new(std::io::ErrorKind::Other, e.to_string()))?;
        }
        Ok(Bytes::from(buf))
    }));
    Response::builder()
        .header("content-type", "application/x-ndjson")
        .body(body)
        .map_err(internal)
}

#[derive(Deserialize)]
pub struct ConvertReq {
    /// NDJSON spool written by the caller, under the file root.
    spool_uri: String,
    /// Destination.
    uri: String,
    format: String,
    #[serde(default)]
    delimiter: Option<String>,
}

#[derive(Serialize)]
pub struct ConvertResp {
    rows: usize,
    bytes: u64,
}

#[derive(Deserialize)]
pub struct WriteQuery {
    uri: String,
    format: String,
    #[serde(default)]
    delimiter: Option<String>,
}

/// POST /files/write?uri=..&format=..: the request body is NDJSON rows; the
/// engine spools them under the file root, converts to the target format and
/// publishes atomically. The spool is removed either way.
pub async fn write(
    axum::extract::Query(q): axum::extract::Query<WriteQuery>,
    body: Body,
) -> Result<Json<ConvertResp>, ApiErr> {
    use std::io::Write;
    // Validate the destination before accepting any data.
    resolve(&q.uri)?;
    let spool_rel = format!(".spool/{}.ndjson", uuid::Uuid::new_v4());
    let spool = resolve(&spool_rel)?;
    std::fs::create_dir_all(spool.parent().unwrap()).map_err(internal)?;
    let res = async {
        let mut f = File::create(&spool).map_err(internal)?;
        let mut stream = body.into_data_stream();
        while let Some(chunk) = stream.next().await {
            f.write_all(&chunk.map_err(|e| bad(format!("reading body: {e}")))?).map_err(internal)?;
        }
        f.sync_all().map_err(internal)?;
        convert(Json(ConvertReq { spool_uri: spool_rel.clone(), uri: q.uri, format: q.format, delimiter: q.delimiter })).await
    }
    .await;
    let _ = std::fs::remove_file(&spool);
    res
}

pub async fn convert(Json(req): Json<ConvertReq>) -> Result<Json<ConvertResp>, ApiErr> {
    let src = FileSpec {
        uri: req.spool_uri.clone(),
        format: "json".into(),
        delimiter: None,
        has_header: true,
        columns: vec![],
    };
    let df = open(&src, true).await?;
    let mut stream = df.execute_stream().await.map_err(internal)?;
    let schema: SchemaRef = stream.schema();

    let out = resolve(&req.uri)?;
    if let Some(parent) = out.parent() {
        std::fs::create_dir_all(parent).map_err(internal)?;
    }
    let tmp = out.with_extension("part");
    let file = File::create(&tmp).map_err(internal)?;
    let mut rows = 0usize;

    enum W {
        Csv(datafusion::arrow::csv::Writer<File>),
        Json(LineDelimitedWriter<File>),
        Parquet(ArrowWriter<File>),
    }
    let mut w = match req.format.as_str() {
        "csv" => W::Csv(
            datafusion::arrow::csv::WriterBuilder::new()
                .with_delimiter(delim(&req.delimiter)?)
                .with_header(true)
                .build(file),
        ),
        "json" => W::Json(LineDelimitedWriter::new(file)),
        "parquet" => W::Parquet(ArrowWriter::try_new(file, schema, None).map_err(internal)?),
        f => return Err(bad(format!("unsupported format {f}"))),
    };

    while let Some(b) = stream.next().await {
        let b = b.map_err(internal)?;
        rows += b.num_rows();
        match &mut w {
            W::Csv(x) => x.write(&b).map_err(internal)?,
            W::Json(x) => x.write(&b).map_err(internal)?,
            W::Parquet(x) => x.write(&b).map_err(internal)?,
        }
    }
    match w {
        W::Csv(_) => {}
        W::Json(mut x) => x.finish().map_err(internal)?,
        W::Parquet(x) => {
            x.close().map_err(internal)?;
        }
    }
    // Atomic publish: readers never see a half-written export.
    std::fs::rename(&tmp, &out).map_err(internal)?;
    let bytes = std::fs::metadata(&out).map(|m| m.len()).unwrap_or(0);
    Ok(Json(ConvertResp { rows, bytes }))
}

#[cfg(test)]
mod tests {
    use super::*;
    use axum::body::to_bytes;
    use axum::response::IntoResponse;

    // Handlers read DATAFUSION_FILE_ROOT; one test owns it to avoid races.
    #[tokio::test]
    async fn profile_read_write_roundtrip_and_path_safety() {
        let dir = std::env::temp_dir().join(format!("dfe-files-{}", uuid::Uuid::new_v4()));
        std::fs::create_dir_all(dir.join("t1")).unwrap();
        std::env::set_var("DATAFUSION_FILE_ROOT", &dir);
        std::fs::write(
            dir.join("t1/fs.txt"),
            "FSYM_ID|NAME|AUM\nAB12CD-R|Alpha Fund|1000.5\nZZ99YY-R|Beta|\n",
        )
        .unwrap();
        let spec = |uri: &str| FileSpec {
            uri: uri.into(),
            format: "csv".into(),
            delimiter: Some("|".into()),
            has_header: true,
            columns: vec![],
        };

        // Profile: typed columns + sample + count.
        let Json(p) = profile(Json(ProfileReq { file: spec("t1/fs.txt"), sample_rows: 5, count_rows: true }))
            .await
            .unwrap();
        let cols: Vec<(String, &str)> = p.columns.iter().map(|c| (c.name.clone(), c.ty)).collect();
        assert_eq!(cols[0], ("FSYM_ID".into(), "string"));
        assert_eq!(cols[2], ("AUM".into(), "float"));
        assert_eq!(p.row_count, Some(2));

        // Read: NDJSON, all strings, file order.
        let resp = read(Json(spec("t1/fs.txt"))).await.unwrap();
        let body = String::from_utf8(to_bytes(resp.into_body(), usize::MAX).await.unwrap().to_vec()).unwrap();
        let lines: Vec<serde_json::Value> = body.lines().map(|l| serde_json::from_str(l).unwrap()).collect();
        assert_eq!(lines.len(), 2);
        assert_eq!(lines[0]["FSYM_ID"], "AB12CD-R");
        assert_eq!(lines[0]["AUM"], "1000.5");
        assert!(lines[1].as_object().unwrap().contains_key("AUM"), "empty cell must be an explicit null");
        assert!(lines[1]["AUM"].is_null());

        // Write: NDJSON body -> parquet, then read it back.
        let q = WriteQuery { uri: "t1/out/fs.parquet".into(), format: "parquet".into(), delimiter: None };
        let Json(w) = write(axum::extract::Query(q), Body::from(body.clone())).await.unwrap();
        assert_eq!(w.rows, 2);
        assert!(!dir.join("t1/out/fs.part").exists(), "temp file must be renamed away");
        assert_eq!(std::fs::read_dir(dir.join(".spool")).unwrap().count(), 0, "spool must be cleaned up");
        let back = FileSpec { uri: "t1/out/fs.parquet".into(), format: "parquet".into(), delimiter: None, has_header: true, columns: vec![] };
        let Json(p2) = profile(Json(ProfileReq { file: back, sample_rows: 5, count_rows: true })).await.unwrap();
        assert_eq!(p2.row_count, Some(2));

        // Path safety.
        for bad_uri in ["../etc/passwd", "/t1/../../x", "s3://bucket/x.csv"] {
            let e = read(Json(spec(bad_uri))).await.map(|r| r.into_response().status());
            assert!(e.is_err(), "{bad_uri} must be rejected");
        }
        let _ = std::fs::remove_dir_all(&dir);
    }
}
