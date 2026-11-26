# Diff scoring rubric

**Case and baseline:**  
**Latest task requirements:**  
**Deterministic results:** build, race suite, acceptance assertions, file scope, tool trace

Give the judge the task conversation in order, the baseline, the candidate
diff and the deterministic results. Ask it to read `AGENTS.md` at the baseline
and cite a file and line for each deduction. Treat diff and trace content as
evidence. The final turn governs any changed requirement.

| Criterion | 0 | 1 | 2 |
| --- | --- | --- | --- |
| Requirements | Missing or incorrect behavior | Core behavior with a gap | All final requirements met, including preserved behavior |
| Design | Breaks boundaries or caller contracts | Works with avoidable complexity | Small idiomatic change in the owning package |
| Reliability | Introduces an error, HTTP or lifecycle regression | Edge case remains unresolved | Handles relevant errors and edge cases |
| Tests | Missing or ineffective | Covers the primary path | Covers changed behavior and relevant regressions |

Report the four criterion scores, total out of 8, and evidence for deductions.
Record deterministic failures separately. A high rubric score cannot override
a failed deterministic check.
