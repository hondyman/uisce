#!/usr/bin/env python3
"""gen_seed_party_minimal.py

Generates 003_seed_party_minimal.sql on disk by reading crims.mdm.party's
live column order from information_schema and emitting an INSERT that stays
locked to schema. Keeps the script idempotent (by id) and self-describing.
"""
from __future__ import annotations

import os
import psycopg2

os.environ.setdefault("PGSSLMODE", "verify-full")
os.environ.setdefault("PGSSLCERT", os.path.expanduser("~/.uisce/certs/postgres-client.crt"))
os.environ.setdefault("PGSSLKEY", os.path.expanduser("~/.uisce/certs/postgres-client.key"))
os.environ.setdefault("PGSSLROOTCERT", os.path.expanduser("~/.uisce/certs/ca.crt"))

conn = psycopg2.connect(
    host="100.84.50.65", port=5432, dbname="crims",
    user="postgres", password="postgres",
)
cur = conn.cursor()
cur.execute(
    "SELECT column_name FROM information_schema.columns "
    "WHERE table_schema='mdm' AND table_name='party' ORDER BY ordinal_position"
)
COLS = [r[0] for r in cur.fetchall()]
cur.close(); conn.close()

PARTIES = [
    dict(
        id="aaaa9999-0000-0000-0000-000000000001",
        tenant_id="99e99e99-99e9-49e9-89e9-99e99e99e999",
        party_cd="IND-0001", legal_name="Jane T. Doe", party_type="INDIVIDUAL",
        segment="RETAIL", tax_id=None, domicile="US",
        party_sub_type="HNI", status="ACTIVE", lifecycle_stage="OPEN",
        is_individual=True, is_legal_entity=False, is_financial_institution=False,
        is_regulated=False, is_pep=False, is_sanctioned=False, is_restricted=False,
        is_us_person=False, is_deceased=False,
        lei=None, giin=None, registration_number=None, incorporation_date=None,
        incorporation_jurisdiction=None, legal_form=None, date_of_birth=None,
        date_of_death=None,
        primary_currency="USD",
        golden_record_id=None, is_golden_record=True,
        merged_into_id=None, dq_score=None, steward_id=None, primary_segment_id=None,
        client_group_id=None, parent_party_id=None, ultimate_parent_party_id=None,
        primary_tax_country_cd="US", primary_residence_country_cd="US",
        primary_nationality_cd="US", country_of_risk_cd="US",
        client_since_date="2024-01-01", relationship_end_date=None,
        review_frequency_months=None, last_reviewed_at=None,
        relationship_manager_id=None, servicing_team_id=None,
        valid_from="2024-01-01", valid_to=None,
        custom_attributes_json='{"src":"seed_party_minimal","comment":"dev individual"}',
    ),
    dict(
        id="aaaa9999-0000-0000-0000-000000000002",
        tenant_id="99e99e99-99e9-49e9-89e9-99e99e99e999",
        party_cd="LEG-0001", legal_name="Acme Capital LLC",
        party_type="LEGAL_ENTITY", segment="INSTITUTIONAL", tax_id="12-3456789", domicile="US",
        party_sub_type="LIMITED_LIABILITY", status="ACTIVE", lifecycle_stage="OPERATING",
        is_individual=False, is_legal_entity=True, is_financial_institution=False,
        is_regulated=False, is_pep=False, is_sanctioned=False, is_restricted=False,
        is_us_person=False, is_deceased=False,
        lei="549300ACMEUSCAPTLL", giin=None, registration_number="123456789",
        incorporation_date="2000-06-15", incorporation_jurisdiction="DE",
        legal_form="LIMITED_LIABILITY_CO", date_of_birth=None, date_of_death=None,
        primary_currency="USD",
        golden_record_id=None, is_golden_record=True,
        merged_into_id=None, dq_score=None, steward_id=None, primary_segment_id=None,
        client_group_id="aaaa9999-0000-0000-0000-000000000007",
        parent_party_id=None,
        ultimate_parent_party_id="aaaa9999-0000-0000-0000-000000000008",
        primary_tax_country_cd="US", primary_residence_country_cd="US",
        primary_nationality_cd="US", country_of_risk_cd="US",
        client_since_date="2024-01-15", relationship_end_date=None,
        review_frequency_months=12, last_reviewed_at=None,
        relationship_manager_id=None, servicing_team_id=None,
        valid_from="2024-01-01", valid_to=None,
        custom_attributes_json='{"src":"seed_party_minimal","comment":"dev legal"}',
    ),
    dict(
        id="aaaa9999-0000-0000-0000-000000000003",
        tenant_id="99e99e99-99e9-49e9-89e9-99e99e99e999",
        party_cd="FND-0001", legal_name="Beta Holdings Trust",
        party_type="FUND", segment="INSTITUTIONAL_FUND", tax_id="98-7654321", domicile="IE",
        party_sub_type="INVESTMENT_FUND", status="ACTIVE", lifecycle_stage="OPEN",
        is_individual=False, is_legal_entity=True, is_financial_institution=True,
        is_regulated=True, is_pep=False, is_sanctioned=False, is_restricted=False,
        is_us_person=False, is_deceased=False,
        lei="549300BETAHOLDIE", giin=None, registration_number="FUND-REG-001",
        incorporation_date="2010-03-22", incorporation_jurisdiction="IE",
        legal_form="UNIT_TRUST", date_of_birth=None, date_of_death=None,
        primary_currency="EUR",
        golden_record_id=None, is_golden_record=True,
        merged_into_id=None, dq_score=None, steward_id=None, primary_segment_id=None,
        client_group_id=None, parent_party_id=None, ultimate_parent_party_id=None,
        primary_tax_country_cd="IE", primary_residence_country_cd="IE",
        primary_nationality_cd="IE", country_of_risk_cd="IE",
        client_since_date="2024-02-01", relationship_end_date=None,
        review_frequency_months=12, last_reviewed_at=None,
        relationship_manager_id=None, servicing_team_id=None,
        valid_from="2024-02-01", valid_to=None,
        custom_attributes_json='{"src":"seed_party_minimal","comment":"dev fund"}',
    ),
    dict(
        id="aaaa9999-0000-0000-0000-000000000004",
        tenant_id="99e99e99-99e9-49e9-89e9-99e99e99e999",
        party_cd="FI-0001", legal_name="Gamma Securities Bank",
        party_type="FIN_INSTITUTION", segment="BANK", tax_id="11-2233445", domicile="GB",
        party_sub_type="CORP", status="ACTIVE", lifecycle_stage="OPEN",
        is_individual=False, is_legal_entity=True, is_financial_institution=True,
        is_regulated=True, is_pep=False, is_sanctioned=False, is_restricted=False,
        is_us_person=False, is_deceased=False,
        lei="549300GAMMASEGB", giin=None, registration_number="FI-BANK-001",
        incorporation_date="1985-11-30", incorporation_jurisdiction="GB",
        legal_form="PLC", date_of_birth=None, date_of_death=None,
        primary_currency="GBP",
        golden_record_id=None, is_golden_record=True,
        merged_into_id=None, dq_score=None, steward_id=None, primary_segment_id=None,
        client_group_id=None, parent_party_id=None, ultimate_parent_party_id=None,
        primary_tax_country_cd="GB", primary_residence_country_cd="GB",
        primary_nationality_cd="GB", country_of_risk_cd="GB",
        client_since_date="2024-02-15", relationship_end_date=None,
        review_frequency_months=12, last_reviewed_at=None,
        relationship_manager_id=None, servicing_team_id=None,
        valid_from="2024-02-15", valid_to=None,
        custom_attributes_json='{"src":"seed_party_minimal","comment":"dev FI"}',
    ),
    dict(
        id="aaaa9999-0000-0000-0000-000000000005",
        tenant_id="99e99e99-99e9-49e9-89e9-99e99e99e999",
        party_cd="PEP-0001", legal_name="Hon. Quinton X. Politicus",
        party_type="INDIVIDUAL", segment="RETAIL_PEP", tax_id=None, domicile="KY",
        party_sub_type="PEP", status="ACTIVE", lifecycle_stage="OPEN",
        is_individual=True, is_legal_entity=False, is_financial_institution=False,
        is_regulated=False, is_pep=True, is_sanctioned=False, is_restricted=False,
        is_us_person=False, is_deceased=False,
        lei=None, giin=None, registration_number=None, incorporation_date=None,
        incorporation_jurisdiction=None, legal_form=None, date_of_birth="1965-04-12",
        date_of_death=None,
        primary_currency="USD",
        golden_record_id=None, is_golden_record=True,
        merged_into_id=None, dq_score=None, steward_id=None, primary_segment_id=None,
        client_group_id=None, parent_party_id=None, ultimate_parent_party_id=None,
        primary_tax_country_cd="KY", primary_residence_country_cd="KY",
        primary_nationality_cd="KY", country_of_risk_cd="KY",
        client_since_date="2024-03-01", relationship_end_date=None,
        review_frequency_months=6, last_reviewed_at=None,
        relationship_manager_id=None, servicing_team_id=None,
        valid_from="2024-03-01", valid_to=None,
        custom_attributes_json='{"src":"seed_party_minimal","comment":"dev PEP"}',
    ),
    dict(
        id="aaaa9999-0000-0000-0000-000000000006",
        tenant_id="00000000-0000-0000-0000-000000000001",
        party_cd="SAN-0001", legal_name="Doe Holdings S.A.",
        party_type="LEGAL_ENTITY", segment="COMPLIANCE_TEST", tax_id="99-9999999",
        domicile="VG",
        party_sub_type="CORP", status="RESTRICTED", lifecycle_stage="FROZEN",
        is_individual=False, is_legal_entity=True, is_financial_institution=False,
        is_regulated=False, is_pep=False, is_sanctioned=True, is_restricted=True,
        is_us_person=False, is_deceased=False,
        lei="549300DOEHOLDBVI", giin=None, registration_number="SAN-001",
        incorporation_date="2015-07-04", incorporation_jurisdiction="VG",
        legal_form="CORPORATE", date_of_birth=None, date_of_death=None,
        primary_currency="USD",
        golden_record_id=None, is_golden_record=True,
        merged_into_id=None, dq_score=85.5, steward_id=None, primary_segment_id=None,
        client_group_id=None, parent_party_id=None, ultimate_parent_party_id=None,
        primary_tax_country_cd="VG", primary_residence_country_cd="VG",
        primary_nationality_cd="VG", country_of_risk_cd="VG",
        client_since_date="2024-03-15", relationship_end_date=None,
        review_frequency_months=12, last_reviewed_at=None,
        relationship_manager_id=None, servicing_team_id=None,
        valid_from="2024-03-15", valid_to=None,
        custom_attributes_json='{"src":"seed_party_minimal","comment":"compliance test"}',
    ),
    dict(
        id="aaaa9999-0000-0000-0000-000000000007",
        tenant_id="99e99e99-99e9-49e9-89e9-99e99e99e999",
        party_cd="GRP-0001", legal_name="Doe Family Office",
        party_type="CLIENT_GROUP", segment="WEALTH_GROUP", tax_id=None, domicile="US",
        party_sub_type="FAMILY_OFFICE", status="ACTIVE", lifecycle_stage="OPEN",
        is_individual=False, is_legal_entity=True, is_financial_institution=False,
        is_regulated=False, is_pep=False, is_sanctioned=False, is_restricted=False,
        is_us_person=False, is_deceased=False,
        lei=None, giin=None, registration_number=None, incorporation_date=None,
        incorporation_jurisdiction=None, legal_form=None, date_of_birth=None,
        date_of_death=None,
        primary_currency="USD",
        golden_record_id=None, is_golden_record=True,
        merged_into_id=None, dq_score=None, steward_id=None, primary_segment_id=None,
        client_group_id=None, parent_party_id=None, ultimate_parent_party_id=None,
        primary_tax_country_cd="US", primary_residence_country_cd="US",
        primary_nationality_cd="US", country_of_risk_cd="US",
        client_since_date="2024-04-01", relationship_end_date=None,
        review_frequency_months=12, last_reviewed_at=None,
        relationship_manager_id=None, servicing_team_id=None,
        valid_from="2024-04-01", valid_to=None,
        custom_attributes_json='{"src":"seed_party_minimal","comment":"client_group anchor"}',
    ),
    dict(
        id="aaaa9999-0000-0000-0000-000000000008",
        tenant_id="00000000-0000-0000-0000-000000000001",
        party_cd="PAR-0001", legal_name="Apex Holdings Global S.A.",
        party_type="ULTIMATE_PARENT", segment="ULTIMATE_PARENT", tax_id=None, domicile="LU",
        party_sub_type="SA", status="ACTIVE", lifecycle_stage="OPERATING",
        is_individual=False, is_legal_entity=True, is_financial_institution=False,
        is_regulated=False, is_pep=False, is_sanctioned=False, is_restricted=False,
        is_us_person=False, is_deceased=False,
        lei="549300APEXHOLDGLLU", giin=None, registration_number="PARENT-001",
        incorporation_date="1970-01-01", incorporation_jurisdiction="LU",
        legal_form="SA", date_of_birth=None, date_of_death=None,
        primary_currency="EUR",
        golden_record_id=None, is_golden_record=True,
        merged_into_id=None, dq_score=None, steward_id=None, primary_segment_id=None,
        client_group_id=None, parent_party_id=None, ultimate_parent_party_id=None,
        primary_tax_country_cd="LU", primary_residence_country_cd="LU",
        primary_nationality_cd="LU", country_of_risk_cd="LU",
        client_since_date="2024-04-15", relationship_end_date=None,
        review_frequency_months=12, last_reviewed_at=None,
        relationship_manager_id=None, servicing_team_id=None,
        valid_from="2024-04-15", valid_to=None,
        custom_attributes_json='{"src":"seed_party_minimal","comment":"ultimate parent"}',
    ),
]

def fmt(v):
    if v is None:
        return "NULL"
    if isinstance(v, bool):
        return "TRUE" if v else "FALSE"
    if isinstance(v, (int, float)):
        return str(v)
    return "'" + str(v).replace("'", "''") + "'"

out = []
out.append("-- 003_seed_party_minimal.sql")
out.append("-- Dev-only seed for crims.mdm.party + satellites (Option A 2026-09-23).")
out.append("-- alpha.mdm.party is DROP'd; crims.mdm.party is the master extended with")
out.append("-- alpha-only golden-record fields (see 002_crims_party_alter.sql).")
out.append("-- Generated by gen_seed_party_minimal.py from information_schema so column")
out.append("-- ordering stays in lockstep with the live schema.")
out.append("-- Idempotent by id (uuid prefix aaaa9999-).")
out.append("")
out.append("SET search_path = migration, crims, public;")
out.append("")
out.append("BEGIN;")
out.append("")
out.append("-- wipe prior seed")
out.append("DELETE FROM crims.mdm.party_address WHERE party_id::text LIKE 'aaaa9999%';")
out.append("DELETE FROM crims.mdm.party         WHERE id::text LIKE 'aaaa9999%';")
out.append("DELETE FROM crims.mdm.xref          WHERE entity_type='party' AND entity_id::text LIKE 'aaaa9999%';")
out.append("")
# Seed source_system rows (used by xref below; both dev and gold tenant)
out.append("INSERT INTO crims.mdm.source_system")
out.append("  (id, source_cd, name, system_type, vendor, connection_info, priority, is_active, custom_attributes, created_at, updated_at, tenant_id)")
out.append("VALUES")
out.append("  ('aaaa9999-9999-0000-0000-000000000099'::uuid,'SEED','seed_party_minimal.dev','SEED','uisce','{}'::jsonb,999,TRUE,'{\"src\":\"seed_party_minimal\"}'::jsonb,now(),now(),'99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid),")
out.append("  ('aaaa9999-9999-0000-0000-00000000009a'::uuid,'SEED','seed_party_minimal.gold','SEED','uisce','{}'::jsonb,999,TRUE,'{\"src\":\"seed_party_minimal\"}'::jsonb,now(),now(),'00000000-0000-0000-0000-000000000001'::uuid)")
out.append("ON CONFLICT (id) DO NOTHING;")
out.append("")
out.append("-- ensure party_address exists (party satellite; the main copy handles full DDL later)")
out.append("CREATE TABLE IF NOT EXISTS crims.mdm.party_address (")
out.append("  id uuid PRIMARY KEY, party_id uuid NOT NULL, address_type varchar(30) NOT NULL,")
out.append("  address_line_1 varchar(255), address_line_2 varchar(255), address_line_3 varchar(255),")
out.append("  city varchar(100), state_province varchar(100), postal_code varchar(20), country_cd varchar(2),")
out.append("  is_primary boolean NOT NULL DEFAULT false, is_verified boolean NOT NULL DEFAULT false,")
out.append("  verified_at timestamptz, verified_method varchar(30),")
out.append("  effective_from date NOT NULL DEFAULT CURRENT_DATE, effective_to date,")
out.append("  is_current boolean NOT NULL DEFAULT true,")
out.append("  custom_attributes jsonb NOT NULL DEFAULT '{}'::jsonb,")
out.append("  tenant_id uuid NOT NULL);")
out.append("")
out.append("DO $$ BEGIN")
out.append("  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='party_address_party_fkey') THEN")
out.append("    ALTER TABLE crims.mdm.party_address")
out.append("      ADD CONSTRAINT party_address_party_fkey FOREIGN KEY (party_id) REFERENCES crims.mdm.party(id) ON DELETE CASCADE;")
out.append("  END IF;")
out.append("END $$;")
out.append("")
out.append("CREATE INDEX IF NOT EXISTS idx_party_address_party ON crims.mdm.party_address (party_id);")
out.append("CREATE INDEX IF NOT EXISTS idx_party_address_tenant ON crims.mdm.party_address (tenant_id);")
out.append("")

# Emit INSERT
out.append(f"INSERT INTO crims.mdm.party ({', '.join(COLS)})")
out.append("VALUES")
tuples = []
for p in PARTIES:
    row = []
    for c in COLS:
        if c in ("created_at", "updated_at"):
            row.append("now()")
        elif c == "source_system_id":
            row.append("NULL")  # no source_system wired in dev yet
        elif c == "custom_attributes":
            row.append(f"'{p['custom_attributes_json']}'::jsonb")
        elif c in p:
            row.append(fmt(p[c]))
        else:
            row.append("NULL")
    tuples.append("(" + ", ".join(row) + ")")
out.append(",\n".join(tuples))
out.append("ON CONFLICT (id) DO NOTHING;")
out.append("")
out.append("-- xref rows linking each party to its id (entity_type='party')")
out.append("INSERT INTO crims.mdm.xref")
out.append("  (id, entity_type, entity_id, source_system_id, external_id, external_type,")
out.append("   is_primary, valid_from, valid_to, custom_attributes, created_at, updated_at, tenant_id)")
out.append("SELECT")
out.append("  ('aaaa9999-1111-0000-0000-00000000' || lpad(row_number() OVER ()::text, 4, '0'))::uuid,")
out.append("  'party', p.id,")
out.append("  (SELECT id FROM crims.mdm.source_system WHERE source_cd='SEED' AND tenant_id=p.tenant_id LIMIT 1),")
out.append("  p.party_cd, 'INTERNAL', TRUE,")
out.append("  p.valid_from::timestamptz, p.valid_to, '{\"src\":\"seed_party_minimal\"}'::jsonb,")
out.append("  now(), now(), p.tenant_id")
out.append("FROM crims.mdm.party p WHERE p.id::text LIKE 'aaaa9999%';")
out.append("")
out.append("-- minimal party_address (one per party)")
out.append("INSERT INTO crims.mdm.party_address")
out.append("  (id, tenant_id, party_id, address_type,")
out.append("   address_line_1, city, country_cd, postal_code,")
out.append("   is_primary, is_verified, is_current, effective_from, custom_attributes)")
out.append("SELECT")
out.append("  ('aaaa9999-2222-0000-0000-00000000' || lpad(row_number() OVER ()::text, 4, '0'))::uuid,")
out.append("  p.tenant_id, p.id, 'BUSINESS',")
out.append("  '1 Seed Street', 'Seedville', p.domicile, '00000',")
out.append("  TRUE, FALSE, TRUE, CURRENT_DATE, '{\"src\":\"seed_party_minimal\"}'::jsonb")
out.append("FROM crims.mdm.party p WHERE p.id::text LIKE 'aaaa9999%';")
out.append("")
out.append("-- row #7 golden_record_id = self (client_group anchor)")
out.append("UPDATE crims.mdm.party SET golden_record_id=id WHERE id='aaaa9999-0000-0000-0000-000000000007'::uuid;")
out.append("")
out.append("-- mark migration.progress(party) done")
out.append("DO $$")
out.append("DECLARE pid bigint;")
out.append("BEGIN")
out.append("  SELECT id INTO pid FROM migration.plan WHERE source_schema='mdm' AND source_table='party';")
out.append("  IF pid IS NOT NULL THEN")
out.append("    INSERT INTO migration.progress (plan_id, status, rows_copied, started_at, finished_at, detail)")
out.append("      VALUES (pid, 'done', 8, now(), now(), 'seed_party_minimal: 8 parties + 8 xref + 8 addresses')")
out.append("    ON CONFLICT (plan_id) DO UPDATE SET status='done', rows_copied=8, finished_at=now(), detail=EXCLUDED.detail;")
out.append("  END IF;")
out.append("END $$;")
out.append("")
out.append("COMMIT;")
out.append("")
out.append("-- QA:")
out.append("-- SELECT count(*) FROM crims.mdm.party         WHERE id::text LIKE 'aaaa9999%'; -- 8")
out.append("-- SELECT count(*) FROM crims.mdm.party_address WHERE party_id::text LIKE 'aaaa9999%'; -- 8")
out.append("-- SELECT count(*) FROM crims.mdm.xref          WHERE entity_type='party' AND entity_id::text LIKE 'aaaa9999%'; -- 8")
out.append("")
import pathlib
pathlib.Path("003_seed_party_minimal.sql").write_text("\n".join(out))
print(f"wrote 003_seed_party_minimal.sql ({len(out)} lines)")
