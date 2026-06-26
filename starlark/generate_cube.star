def generate_cube(bo):
    cube = {
        "cube": bo["name"],
        "sql_table": bo.get("sql_table", bo["name"]),
        "measures": {},
        "dimensions": {}
    }

    for term in bo["semantic_terms"]:
        if term["type"] == "measure":
            cube["measures"][term["id"].split('.')[-1]] = {
                "type": "sum",
                "sql": 'semantic("%s")' % term["id"],
                "meta": { "materialization": term.get("materialization", "virtual") }
            }
        else:
            cube["dimensions"][term["id"].split('.')[-1]] = {
                "sql": 'semantic("%s")' % term["id"],
                "type": term.get("data_type", "string")
            }

    return cube
