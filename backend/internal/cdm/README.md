# FINOS CDM Go Distribution

This directory is intended to hold the generated Go code from the [FINOS Common Domain Model](https://github.com/finos/common-domain-model).

## Instructions

1.  Download the "CDM as Go" distribution from the FINOS download page or generating it from source.
2.  Extract the Go files into this directory (`backend/internal/cdm`).
3.  Ensure the package name is `cdm` (or adjust the generator configuration if it differs).

The `cdm-generator` tool parses files in this directory to generate the internal platform catalog.
