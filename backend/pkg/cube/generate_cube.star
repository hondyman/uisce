def generate_cube(bo):
    cube = {
        "cube": bo["name"],
        "sql_table": bo.get("sql_table", "iceberg.analytics.%s" % bo["name"]), # Trino/Iceberg naming convention
        "measures": {},
        "dimensions": {}
    }

    # Standard Semantic Terms
    for term in bo.get("semantic_terms", []):
        name = term["id"].split('.')[-1]
        if term["type"] == "measure":
            cube["measures"][name] = {
                "type": "sum",
                "sql": 'semantic("%s")' % term["id"],
                "meta": { "materialization": term.get("materialization", "virtual") }
            }
        else:
            cube["dimensions"][name] = {
                "sql": 'semantic("%s")' % term["id"],
                "type": term.get("data_type", "string")
            }

    # Calculated Fields (Phase 9: Workday Killer)
    for calc in bo.get("calc_fields", []):
        name = calc["name"]
        if calc.get("is_measure", True):
            cube["measures"][name] = {
                "type": "number",
                "sql": calc["sql_expr"],
                "meta": { "realtime": calc.get("realtime", True) }
            }
        else:
            cube["dimensions"][name] = {
                "sql": calc["sql_expr"],
                "type": calc.get("data_type", "string")
            }

    return cube
