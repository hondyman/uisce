"""Iceberg/Trino feature value loader"""

import logging
from datetime import datetime
from typing import Tuple
import numpy as np
import pyarrow.parquet as pq
import pyarrow.dataset as ds

from app.config import settings

logger = logging.getLogger(__name__)

async def load_feature_values(
    feature_id: str,
    start_time: datetime,
    end_time: datetime,
    tenant_id: str,
    region: str
) -> np.ndarray:
    """
    Load feature values from Iceberg table for a time window.
    
    Iceberg table naming: features.<namespace>.<feature_name>_v<version>
    Partitioned by: feature_date, tenant_id, region
    """
    try:
        # Parse feature ID to get namespace and name
        # feature:orders.revenue_v1 -> namespace=orders, name=revenue_v1
        if not feature_id.startswith("feature:"):
            raise ValueError(f"Invalid feature_id format: {feature_id}")
        
        namespace, name = feature_id[8:].rsplit(".", 1)  # Remove "feature:" prefix
        
        table_name = f"features.{namespace}.{name}"
        
        # Load from Iceberg using Trino/PyArrow
        dataset = ds.dataset(
            f"iceberg://{settings.TRINO_CATALOG}/{table_name}",
            format="parquet"
        )
        
        # Filter by time window and tenant/region
        filtered = dataset.to_table(
            filters=[
                ("feature_date", ">=", start_time.date()),
                ("feature_date", "<=", end_time.date()),
                ("tenant_id", "==", tenant_id),
                ("region", "==", region)
            ]
        )
        
        # Extract feature values column
        if "value" in filtered.column_names:
            values = filtered["value"].to_numpy()
        elif "feature_value" in filtered.column_names:
            values = filtered["feature_value"].to_numpy()
        else:
            # Try first numeric column
            for col in filtered.column_names:
                if col not in ["feature_date", "tenant_id", "region", "feature_id"]:
                    values = filtered[col].to_numpy()
                    break
        
        logger.info(f"Loaded {len(values)} values for {feature_id} from {start_time} to {end_time}")
        
        return values
        
    except Exception as e:
        logger.error(f"Failed to load feature values for {feature_id}: {str(e)}")
        raise

async def load_feature_values_from_path(path: str) -> np.ndarray:
    """Load feature values from a Parquet file path"""
    try:
        table = pq.read_table(path)
        
        # Try to find a numeric column
        for col in table.column_names:
            if col not in ["feature_date", "tenant_id", "region", "feature_id"]:
                return table[col].to_numpy()
        
        raise ValueError(f"No numeric columns found in {path}")
    except Exception as e:
        logger.error(f"Failed to load from path {path}: {str(e)}")
        raise
