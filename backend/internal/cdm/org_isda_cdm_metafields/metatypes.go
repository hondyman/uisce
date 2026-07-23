/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm_metafields

type FieldWithMeta struct {
	Value interface{}
	Meta  MetaFields
}

type ReferenceWithMeta struct {
	GlobalReference   string
	ExternalReference string
	Address           Reference
	Value             interface{}
}

type MetaFields struct {
	Scheme      string
	Location    string
	GlobalKey   string
	ExternalKey string
}

type MetaAndTemplateFields struct {
	Scheme                  string
	Location                string
	GlobalKey               string
	ExternalKey             string
	TemplateGlobalReference string
}

type Key struct {
	Scope string
	Value string
}

type Reference struct {
	Scope string
	Value string
}
