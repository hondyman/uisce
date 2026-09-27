-- 002c_rating_fixup.sql (v3)
-- Apply FKs without rating_scale_map reference (added separately).
SET search_path = migration, crims, public;
-- This is the historical 002c file (already applied to live crims).
-- DO NOT re-run: it DROPs the rating reference tables and recreates them.
-- For a rerunnable version, use 0015_mdm_rating_reference_crims.up.sql
-- 002c_rating_fixup.sql (v3)
-- Apply FKs without rating_scale_map reference (added separately).
SET search_path = migration, crims, public;
BEGIN;
DROP TABLE IF EXISTS crims.mdm.rating_scale CASCADE;
DROP TABLE IF EXISTS crims.mdm.rating_agency CASCADE;
DROP TABLE IF EXISTS crims.mdm.rating_outlook CASCADE;
DROP TABLE IF EXISTS crims.mdm.rating_watch CASCADE;
DROP TABLE IF EXISTS crims.mdm.rating_type CASCADE;
DROP TABLE IF EXISTS crims.mdm.rating_action_type CASCADE;

-- === alpha.mdm.rating_scale DDL (FK-stripped) ===
--
-- PostgreSQL database dump
--

\restrict eikPxzu4eRroMbki39VthqwIpFb83ifvhcXo0RZDuLQUEksDsZ70dJob8fgAs4e

-- Dumped from database version 18.6 (Ubuntu 18.6-1.pgdg24.04+2)
-- Dumped by pg_dump version 18.6

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: rating_scale; Type: TABLE; Schema: mdm; Owner: -
--

CREATE TABLE mdm.rating_scale (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    agency_id uuid,
    agency_cd character varying(30),
    scale_cd character varying(50) NOT NULL,
    scale_name character varying(150),
    agency_scale_type character varying(30),
    value character varying(20) NOT NULL,
    rank_no integer,
    rating_category character varying(20),
    is_not_rated boolean DEFAULT false,
    is_withdrawn boolean DEFAULT false,
    is_na boolean DEFAULT false,
    display_order integer DEFAULT 0,
    effective_from date DEFAULT CURRENT_DATE,
    effective_to date,
    is_active boolean DEFAULT true,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL
);

ALTER TABLE ONLY mdm.rating_scale FORCE ROW LEVEL SECURITY;


--
-- Name: rating_scale rating_scale_pkey; Type: CONSTRAINT; Schema: mdm; Owner: -
--

ALTER TABLE ONLY mdm.rating_scale
    ADD CONSTRAINT rating_scale_pkey PRIMARY KEY (id);


--
-- Name: rating_scale rating_scale_scale_value_key; Type: CONSTRAINT; Schema: mdm; Owner: -
--

ALTER TABLE ONLY mdm.rating_scale
    ADD CONSTRAINT rating_scale_scale_value_key UNIQUE (tenant_id, scale_cd, value);


--
-- Name: idx_rs_agency_id; Type: INDEX; Schema: mdm; Owner: -
--

CREATE INDEX idx_rs_agency_id ON mdm.rating_scale USING btree (agency_id) WHERE (agency_id IS NOT NULL);


--
-- Name: idx_rs_category; Type: INDEX; Schema: mdm; Owner: -
--

CREATE INDEX idx_rs_category ON mdm.rating_scale USING btree (rating_category) WHERE (rating_category IS NOT NULL);


--
-- Name: rating_scale fk_rs_agency; Type: FK CONSTRAINT; Schema: mdm; Owner: -
--

--
-- Name: rating_scale; Type: ROW SECURITY; Schema: mdm; Owner: -
--

ALTER TABLE mdm.rating_scale ENABLE ROW LEVEL SECURITY;

--
-- Name: rating_scale rating_scale_tenant_read; Type: POLICY; Schema: mdm; Owner: -
--

CREATE POLICY rating_scale_tenant_read ON mdm.rating_scale FOR SELECT USING (((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid) OR (tenant_id = COALESCE((current_setting('app.shared_reference_tenant'::text, true))::uuid, '00000000-0000-0000-0000-000000000001'::uuid))));


--
-- Name: rating_scale rating_scale_tenant_write; Type: POLICY; Schema: mdm; Owner: -
--

CREATE POLICY rating_scale_tenant_write ON mdm.rating_scale USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)) WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));


--
-- PostgreSQL database dump complete
--

\unrestrict eikPxzu4eRroMbki39VthqwIpFb83ifvhcXo0RZDuLQUEksDsZ70dJob8fgAs4e



-- === alpha.mdm.rating_agency DDL (FK-stripped) ===
--
-- PostgreSQL database dump
--

\restrict mCRBrmiV28DNH5ENdoNVi5J83jGuUApMHhy6Twmber5R2lon6Ln5XrfkaBAX0Fj

-- Dumped from database version 18.6 (Ubuntu 18.6-1.pgdg24.04+2)
-- Dumped by pg_dump version 18.6

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: rating_agency; Type: TABLE; Schema: mdm; Owner: -
--

CREATE TABLE mdm.rating_agency (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    agency_cd character varying(30) NOT NULL,
    name character varying(250) NOT NULL,
    short_name character varying(50),
    agency_type character varying(30) NOT NULL,
    domicile character varying(2),
    parent_agency_id uuid,
    is_nrsro boolean DEFAULT false NOT NULL,
    is_ecai boolean DEFAULT false NOT NULL,
    is_naic_acceptable boolean DEFAULT false NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    registration_number character varying(50),
    registration_authority character varying(100),
    website character varying(250),
    status character varying(20) DEFAULT 'ACTIVE'::character varying NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT chk_ra_status CHECK (((status)::text = ANY ((ARRAY['ACTIVE'::character varying, 'DORMANT'::character varying, 'WITHDRAWN'::character varying, 'SUSPENDED'::character varying])::text[]))),
    CONSTRAINT chk_ra_type CHECK (((agency_type)::text = ANY ((ARRAY['NRSRO'::character varying, 'ECAI'::character varying, 'NATIONAL'::character varying, 'REGIONAL'::character varying, 'INTERNAL'::character varying, 'MODEL'::character varying, 'MARKET_IMPLIED'::character varying])::text[])))
);

ALTER TABLE ONLY mdm.rating_agency FORCE ROW LEVEL SECURITY;


--
-- Name: rating_agency rating_agency_cd_key; Type: CONSTRAINT; Schema: mdm; Owner: -
--

ALTER TABLE ONLY mdm.rating_agency
    ADD CONSTRAINT rating_agency_cd_key UNIQUE (tenant_id, agency_cd);


--
-- Name: rating_agency rating_agency_pkey; Type: CONSTRAINT; Schema: mdm; Owner: -
--

ALTER TABLE ONLY mdm.rating_agency
    ADD CONSTRAINT rating_agency_pkey PRIMARY KEY (id);


--
-- Name: idx_ra_active; Type: INDEX; Schema: mdm; Owner: -
--

CREATE INDEX idx_ra_active ON mdm.rating_agency USING btree (is_active, is_nrsro);


--
-- Name: idx_ra_tenant; Type: INDEX; Schema: mdm; Owner: -
--

CREATE INDEX idx_ra_tenant ON mdm.rating_agency USING btree (tenant_id);


--
-- Name: idx_ra_type; Type: INDEX; Schema: mdm; Owner: -
--

CREATE INDEX idx_ra_type ON mdm.rating_agency USING btree (agency_type);


--
-- Name: rating_agency fk_ra_parent; Type: FK CONSTRAINT; Schema: mdm; Owner: -
--

--
-- Name: rating_agency; Type: ROW SECURITY; Schema: mdm; Owner: -
--

ALTER TABLE mdm.rating_agency ENABLE ROW LEVEL SECURITY;

--
-- Name: rating_agency rating_agency_tenant_read; Type: POLICY; Schema: mdm; Owner: -
--

CREATE POLICY rating_agency_tenant_read ON mdm.rating_agency FOR SELECT USING (((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid) OR (tenant_id = COALESCE((current_setting('app.shared_reference_tenant'::text, true))::uuid, '00000000-0000-0000-0000-000000000001'::uuid))));


--
-- Name: rating_agency rating_agency_tenant_write; Type: POLICY; Schema: mdm; Owner: -
--

CREATE POLICY rating_agency_tenant_write ON mdm.rating_agency USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)) WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));


--
-- PostgreSQL database dump complete
--

\unrestrict mCRBrmiV28DNH5ENdoNVi5J83jGuUApMHhy6Twmber5R2lon6Ln5XrfkaBAX0Fj



-- === alpha.mdm.rating_outlook DDL (FK-stripped) ===
--
-- PostgreSQL database dump
--

\restrict zQAnJUl4IAhlVRu8fD9i62cdEnr5yN8kjkNMf6Q1LDHY5CVLW6YgTGzHFrXLgVt

-- Dumped from database version 18.6 (Ubuntu 18.6-1.pgdg24.04+2)
-- Dumped by pg_dump version 18.6

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: rating_outlook; Type: TABLE; Schema: mdm; Owner: -
--

CREATE TABLE mdm.rating_outlook (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    outlook_cd character varying(20) NOT NULL,
    name character varying(50) NOT NULL,
    direction character varying(20) NOT NULL,
    horizon_months integer,
    is_active boolean DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT chk_ro_direction CHECK (((direction)::text = ANY ((ARRAY['POSITIVE'::character varying, 'NEGATIVE'::character varying, 'STABLE'::character varying, 'DEVELOPING'::character varying, 'NEUTRAL'::character varying])::text[])))
);

ALTER TABLE ONLY mdm.rating_outlook FORCE ROW LEVEL SECURITY;


--
-- Name: rating_outlook rating_outlook_cd_key; Type: CONSTRAINT; Schema: mdm; Owner: -
--

ALTER TABLE ONLY mdm.rating_outlook
    ADD CONSTRAINT rating_outlook_cd_key UNIQUE (tenant_id, outlook_cd);


--
-- Name: rating_outlook rating_outlook_pkey; Type: CONSTRAINT; Schema: mdm; Owner: -
--

ALTER TABLE ONLY mdm.rating_outlook
    ADD CONSTRAINT rating_outlook_pkey PRIMARY KEY (id);


--
-- Name: idx_ro_tenant; Type: INDEX; Schema: mdm; Owner: -
--

CREATE INDEX idx_ro_tenant ON mdm.rating_outlook USING btree (tenant_id);


--
-- Name: rating_outlook; Type: ROW SECURITY; Schema: mdm; Owner: -
--

ALTER TABLE mdm.rating_outlook ENABLE ROW LEVEL SECURITY;

--
-- Name: rating_outlook rating_outlook_tenant_read; Type: POLICY; Schema: mdm; Owner: -
--

CREATE POLICY rating_outlook_tenant_read ON mdm.rating_outlook FOR SELECT USING (((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid) OR (tenant_id = COALESCE((current_setting('app.shared_reference_tenant'::text, true))::uuid, '00000000-0000-0000-0000-000000000001'::uuid))));


--
-- Name: rating_outlook rating_outlook_tenant_write; Type: POLICY; Schema: mdm; Owner: -
--

CREATE POLICY rating_outlook_tenant_write ON mdm.rating_outlook USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)) WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));


--
-- PostgreSQL database dump complete
--

\unrestrict zQAnJUl4IAhlVRu8fD9i62cdEnr5yN8kjkNMf6Q1LDHY5CVLW6YgTGzHFrXLgVt



-- === alpha.mdm.rating_watch DDL (FK-stripped) ===
--
-- PostgreSQL database dump
--

\restrict gk7VGhjy5OaGWDZGEocjx9JCj5PDYdglsQJArK4d426pQPCVyqclTQRGlWv0Jef

-- Dumped from database version 18.6 (Ubuntu 18.6-1.pgdg24.04+2)
-- Dumped by pg_dump version 18.6

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: rating_watch; Type: TABLE; Schema: mdm; Owner: -
--

CREATE TABLE mdm.rating_watch (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    watch_cd character varying(30) NOT NULL,
    name character varying(50) NOT NULL,
    direction character varying(20) NOT NULL,
    horizon_days integer,
    is_active boolean DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT chk_rw_direction CHECK (((direction)::text = ANY ((ARRAY['POSITIVE'::character varying, 'NEGATIVE'::character varying, 'DEVELOPING'::character varying, 'EVOLVING'::character varying])::text[])))
);

ALTER TABLE ONLY mdm.rating_watch FORCE ROW LEVEL SECURITY;


--
-- Name: rating_watch rating_watch_cd_key; Type: CONSTRAINT; Schema: mdm; Owner: -
--

ALTER TABLE ONLY mdm.rating_watch
    ADD CONSTRAINT rating_watch_cd_key UNIQUE (tenant_id, watch_cd);


--
-- Name: rating_watch rating_watch_pkey; Type: CONSTRAINT; Schema: mdm; Owner: -
--

ALTER TABLE ONLY mdm.rating_watch
    ADD CONSTRAINT rating_watch_pkey PRIMARY KEY (id);


--
-- Name: idx_rw_tenant; Type: INDEX; Schema: mdm; Owner: -
--

CREATE INDEX idx_rw_tenant ON mdm.rating_watch USING btree (tenant_id);


--
-- Name: rating_watch; Type: ROW SECURITY; Schema: mdm; Owner: -
--

ALTER TABLE mdm.rating_watch ENABLE ROW LEVEL SECURITY;

--
-- Name: rating_watch rating_watch_tenant_read; Type: POLICY; Schema: mdm; Owner: -
--

CREATE POLICY rating_watch_tenant_read ON mdm.rating_watch FOR SELECT USING (((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid) OR (tenant_id = COALESCE((current_setting('app.shared_reference_tenant'::text, true))::uuid, '00000000-0000-0000-0000-000000000001'::uuid))));


--
-- Name: rating_watch rating_watch_tenant_write; Type: POLICY; Schema: mdm; Owner: -
--

CREATE POLICY rating_watch_tenant_write ON mdm.rating_watch USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)) WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));


--
-- PostgreSQL database dump complete
--

\unrestrict gk7VGhjy5OaGWDZGEocjx9JCj5PDYdglsQJArK4d426pQPCVyqclTQRGlWv0Jef



-- === alpha.mdm.rating_type DDL (FK-stripped) ===
--
-- PostgreSQL database dump
--

\restrict 7LuGBqdbWpXWo1QCtgWVdrg6qngzdEX1Bly9lPcGyN55WY9bvLKhO852myRSrNc

-- Dumped from database version 18.6 (Ubuntu 18.6-1.pgdg24.04+2)
-- Dumped by pg_dump version 18.6

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: rating_type; Type: TABLE; Schema: mdm; Owner: -
--

CREATE TABLE mdm.rating_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    type_cd character varying(50) NOT NULL,
    name character varying(150) NOT NULL,
    applies_to character varying(30) NOT NULL,
    rating_basis character varying(30),
    description text,
    is_active boolean DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT chk_rt_applies CHECK (((applies_to)::text = ANY ((ARRAY['ISSUER'::character varying, 'ISSUE'::character varying, 'TRANCHES'::character varying, 'COUNTERPARTY'::character varying, 'SOVEREIGN'::character varying, 'SUB_SOVEREIGN'::character varying, 'FINANCIAL_STRENGTH'::character varying, 'INSURER'::character varying, 'BANK'::character varying, 'FUND'::character varying])::text[])))
);

ALTER TABLE ONLY mdm.rating_type FORCE ROW LEVEL SECURITY;


--
-- Name: rating_type rating_type_cd_key; Type: CONSTRAINT; Schema: mdm; Owner: -
--

ALTER TABLE ONLY mdm.rating_type
    ADD CONSTRAINT rating_type_cd_key UNIQUE (tenant_id, type_cd);


--
-- Name: rating_type rating_type_pkey; Type: CONSTRAINT; Schema: mdm; Owner: -
--

ALTER TABLE ONLY mdm.rating_type
    ADD CONSTRAINT rating_type_pkey PRIMARY KEY (id);


--
-- Name: idx_rt_tenant; Type: INDEX; Schema: mdm; Owner: -
--

CREATE INDEX idx_rt_tenant ON mdm.rating_type USING btree (tenant_id);


--
-- Name: rating_type; Type: ROW SECURITY; Schema: mdm; Owner: -
--

ALTER TABLE mdm.rating_type ENABLE ROW LEVEL SECURITY;

--
-- Name: rating_type rating_type_tenant_read; Type: POLICY; Schema: mdm; Owner: -
--

CREATE POLICY rating_type_tenant_read ON mdm.rating_type FOR SELECT USING (((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid) OR (tenant_id = COALESCE((current_setting('app.shared_reference_tenant'::text, true))::uuid, '00000000-0000-0000-0000-000000000001'::uuid))));


--
-- Name: rating_type rating_type_tenant_write; Type: POLICY; Schema: mdm; Owner: -
--

CREATE POLICY rating_type_tenant_write ON mdm.rating_type USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)) WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));


--
-- PostgreSQL database dump complete
--

\unrestrict 7LuGBqdbWpXWo1QCtgWVdrg6qngzdEX1Bly9lPcGyN55WY9bvLKhO852myRSrNc



-- === alpha.mdm.rating_action_type DDL (FK-stripped) ===
--
-- PostgreSQL database dump
--

\restrict NLZihEfui2VmZVjpmCmSXKAZjLbHExPS83BFiepXaxKecK6STPUYWKkdcFPZpUy

-- Dumped from database version 18.6 (Ubuntu 18.6-1.pgdg24.04+2)
-- Dumped by pg_dump version 18.6

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: rating_action_type; Type: TABLE; Schema: mdm; Owner: -
--

CREATE TABLE mdm.rating_action_type (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    action_cd character varying(30) NOT NULL,
    name character varying(100) NOT NULL,
    direction character varying(20) NOT NULL,
    is_credit_event boolean DEFAULT false NOT NULL,
    is_active boolean DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT chk_rat_direction CHECK (((direction)::text = ANY ((ARRAY['UPGRADE'::character varying, 'DOWNGRADE'::character varying, 'AFFIRMATION'::character varying, 'INITIAL'::character varying, 'WITHDRAWN'::character varying, 'PLACED_ON_WATCH'::character varying, 'REMOVED_FROM_WATCH'::character varying, 'DEFAULT'::character varying, 'CURE'::character varying, 'OTHER'::character varying])::text[])))
);

ALTER TABLE ONLY mdm.rating_action_type FORCE ROW LEVEL SECURITY;


--
-- Name: rating_action_type rating_action_type_cd_key; Type: CONSTRAINT; Schema: mdm; Owner: -
--

ALTER TABLE ONLY mdm.rating_action_type
    ADD CONSTRAINT rating_action_type_cd_key UNIQUE (tenant_id, action_cd);


--
-- Name: rating_action_type rating_action_type_pkey; Type: CONSTRAINT; Schema: mdm; Owner: -
--

ALTER TABLE ONLY mdm.rating_action_type
    ADD CONSTRAINT rating_action_type_pkey PRIMARY KEY (id);


--
-- Name: idx_rat_tenant; Type: INDEX; Schema: mdm; Owner: -
--

CREATE INDEX idx_rat_tenant ON mdm.rating_action_type USING btree (tenant_id);


--
-- Name: rating_action_type; Type: ROW SECURITY; Schema: mdm; Owner: -
--

ALTER TABLE mdm.rating_action_type ENABLE ROW LEVEL SECURITY;

--
-- Name: rating_action_type rating_action_type_tenant_read; Type: POLICY; Schema: mdm; Owner: -
--

CREATE POLICY rating_action_type_tenant_read ON mdm.rating_action_type FOR SELECT USING (((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid) OR (tenant_id = COALESCE((current_setting('app.shared_reference_tenant'::text, true))::uuid, '00000000-0000-0000-0000-000000000001'::uuid))));


--
-- Name: rating_action_type rating_action_type_tenant_write; Type: POLICY; Schema: mdm; Owner: -
--

CREATE POLICY rating_action_type_tenant_write ON mdm.rating_action_type USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)) WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));


--
-- PostgreSQL database dump complete
--

\unrestrict NLZihEfui2VmZVjpmCmSXKAZjLbHExPS83BFiepXaxKecK6STPUYWKkdcFPZpUy



ALTER TABLE crims.mdm.rating_scale ADD CONSTRAINT fk_rs_agency FOREIGN KEY (agency_id) REFERENCES crims.mdm.rating_agency(id);
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname='mdm' AND tablename='rating_scale' AND policyname='rating_scale_tenant_read') THEN
    CREATE POLICY rating_scale_tenant_read ON crims.mdm.rating_scale
      FOR SELECT USING (tenant_id::text = current_setting('app.tenant_id')::text);
  END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_rating_scale_tenant ON crims.mdm.rating_scale (tenant_id);
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname='mdm' AND tablename='rating_agency' AND policyname='rating_agency_tenant_read') THEN
    CREATE POLICY rating_agency_tenant_read ON crims.mdm.rating_agency
      FOR SELECT USING (tenant_id::text = current_setting('app.tenant_id')::text);
  END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_rating_agency_tenant ON crims.mdm.rating_agency (tenant_id);
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname='mdm' AND tablename='rating_outlook' AND policyname='rating_outlook_tenant_read') THEN
    CREATE POLICY rating_outlook_tenant_read ON crims.mdm.rating_outlook
      FOR SELECT USING (tenant_id::text = current_setting('app.tenant_id')::text);
  END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_rating_outlook_tenant ON crims.mdm.rating_outlook (tenant_id);
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname='mdm' AND tablename='rating_watch' AND policyname='rating_watch_tenant_read') THEN
    CREATE POLICY rating_watch_tenant_read ON crims.mdm.rating_watch
      FOR SELECT USING (tenant_id::text = current_setting('app.tenant_id')::text);
  END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_rating_watch_tenant ON crims.mdm.rating_watch (tenant_id);
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname='mdm' AND tablename='rating_type' AND policyname='rating_type_tenant_read') THEN
    CREATE POLICY rating_type_tenant_read ON crims.mdm.rating_type
      FOR SELECT USING (tenant_id::text = current_setting('app.tenant_id')::text);
  END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_rating_type_tenant ON crims.mdm.rating_type (tenant_id);
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname='mdm' AND tablename='rating_action_type' AND policyname='rating_action_type_tenant_read') THEN
    CREATE POLICY rating_action_type_tenant_read ON crims.mdm.rating_action_type
      FOR SELECT USING (tenant_id::text = current_setting('app.tenant_id')::text);
  END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_rating_action_type_tenant ON crims.mdm.rating_action_type (tenant_id);
COMMIT;
