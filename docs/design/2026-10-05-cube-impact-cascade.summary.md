# Cube impact + cascade — summary

**Status:** Approved · start at **A1**  
**Full:** `docs/design/2026-10-05-cube-impact-cascade.md`

| | |
|--|--|
| Assess | Composition + consumers; `GET /impact`, `POST /impact/preview` |
| Apply | `POST /cascade` + confirmToken; fail closed by default |
| Break detector | Reuse `DetectCubeContractBreaking` |
| UI | Catalog + designer Impact panel (`cubes.ImpactPanel`) |
| PR order | A1 inventory → A2 preview → A3 UI → A4 archive cascade → A5 publish rewire |
| Later | Lakekeeper OAuth (B), extract-N→staging (C), pagestudio delete (D) |
