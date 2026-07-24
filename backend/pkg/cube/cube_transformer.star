# Cube Transformer Template
# This Starlark script can be used to apply advanced transformations to generated Cube models

# Input: model (dict with 'cubes' key)
# Output: result (transformed model)

def add_yoy_measure(cube, measure_name, source_measure):
    """Add a Year-over-Year growth measure based on an existing measure."""
    cube["measures"].append({
        "name": measure_name + "_yoy",
        "sql": "({{{measure_name}}} - LAG({{{measure_name}}}, 1) OVER (ORDER BY created_at)) / NULLIF(LAG({{{measure_name}}}, 1) OVER (ORDER BY created_at), 0) * 100".replace("{{{measure_name}}}", source_measure),
        "type": "number",
        "title": source_measure.replace("_", " ").title() + " YoY Growth %",
        "description": "Year-over-year growth for " + source_measure
    })
    return cube

def add_rolling_avg_measure(cube, measure_name, source_measure, window_days=30):
    """Add a rolling average measure."""
    cube["measures"].append({
        "name": measure_name + "_rolling_avg_" + str(window_days) + "d",
        "sql": "AVG({{{measure_name}}}) OVER (ORDER BY created_at ROWS BETWEEN {window} PRECEDING AND CURRENT ROW)".replace("{{{measure_name}}}", source_measure).replace("{window}", str(window_days)),
        "type": "number",
        "title": source_measure.replace("_", " ").title() + " " + str(window_days) + "-Day Avg",
        "description": str(window_days) + "-day rolling average for " + source_measure
    })
    return cube

def add_preaggregation(cube, name, measures, dimensions, granularity="day"):
    """Add a pre-aggregation configuration to a cube."""
    if "pre_aggregations" not in cube or cube["pre_aggregations"] == None:
        cube["pre_aggregations"] = []
    
    cube["pre_aggregations"].append({
        "name": name,
        "measures": measures,
        "dimensions": dimensions,
        "granularity": granularity
    })
    return cube

# Main transformation logic
def transform(model):
    """
    Apply transformations to the cube model.
    Customize this function for your specific needs.
    """
    for cube in model.get("cubes", []):
        # Example: Add YoY measure for all number measures
        for measure in list(cube.get("measures", [])):
            if measure.get("type") == "number":
                # Uncomment to enable:
                # cube = add_yoy_measure(cube, measure["name"], measure["name"])
                pass
        
        # Example: Add pre-aggregation for measures
        measure_names = [m["name"] for m in cube.get("measures", [])]
        dimension_names = [d["name"] for d in cube.get("dimensions", []) if d.get("type") == "time"]
        
        if measure_names and dimension_names:
            # Uncomment to enable:
            # cube = add_preaggregation(cube, "main", measure_names, dimension_names, "day")
            pass
    
    return model

# Execute transformation
result = transform(ctx.record)
