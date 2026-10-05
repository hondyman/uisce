# Cube impact + cascade — summary

**Status:** A0–A4 **Done** · next **A5** publish_version Confirm  
**Full:** `docs/design/2026-10-05-cube-impact-cascade.md`

| | |
|--|--|
| Assess | Composition + consumers; `GET /impact`, `POST /impact/preview` |
| Apply | `POST /cascade` + confirmToken; fail closed by default |
| Break detector | Reuse `DetectCubeContractBreaking` |
| UI | Catalog + designer Impact panel (`cubes.ImpactPanel`) + archive Confirm |
| PR order | A1–A4 done → **A5** publish rewire |
| Later | Lakekeeper OAuth (B), extract-N→staging (C), pagestudio delete (D) |
