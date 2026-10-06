# Cube impact + cascade — summary

**Status:** Track A **A0–A5 Done**  
**Full:** `docs/design/2026-10-05-cube-impact-cascade.md`

| | |
|--|--|
| Assess | Composition + consumers; `GET /impact`, `POST /impact/preview` |
| Apply | `POST /cascade` + confirmToken; fail closed by default |
| Break detector | Reuse `DetectCubeContractBreaking` |
| UI | Catalog + designer Impact panel (`cubes.ImpactPanel`) + archive/publish Confirm |
| PR order | A1–A5 done (inventory → preview → UI → archive cascade → publish rewire) |
| Later | Lakekeeper OAuth (B), extract-N→staging (C), pagestudio delete (D) |
