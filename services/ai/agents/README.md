# Civic Agent Network

A small, single-purpose multi-agent system for evidence-grounded civic
reasoning. Lives under `services/ai/agents/` alongside the AI gateway.

## Agents

| Agent                 | Module            | Job                                                                 |
|-----------------------|-------------------|---------------------------------------------------------------------|
| `ResearchAgent`       | `researcher.py`   | Retrieve evidence chunks for a question (no model call).            |
| `HistoricalComparator`| `comparator.py`   | Find comparable past bills/scenarios (no model call).              |
| `AssumptionAuditor`   | `auditor.py`      | Flag unsupported factual assertions in a draft answer.             |
| `ScenarioSynthesizer` | `synthesizer.py`  | Compose a multi-paragraph evidence-grounded explanation.            |

All four are registered in `services/ai/main.py` (or a bootstrap module) at
startup; consumers look them up by name via `get_registry().get(name)`.

## Hard invariants

1. **No agent writes to canonical civic truth.** The `Permission` enum does
   not even have a `WRITE_CIVIC_TRUTH` flag; the registry refuses to register
   any agent that claims it. Canonical state (`legislation.*`,
   `government.*`) is mutated exclusively by the Go `services/intelligence`
   service consuming NATS events.
2. **Every output is marked `ai_generated=True`.** The `BaseAgent.run`
   wrapper sets this; subclasses cannot unset it.
3. **Every agent respects a token/cost budget.** Model-calling agents
   (`AssumptionAuditor` with `use_model_for_extraction=True`,
   `ScenarioSynthesizer`) call `Budget.would_exceed` BEFORE the model and
   `Budget.charge` AFTER it. Pre-flight and post-charge are both enforced.
4. **Every agent logs its actions.** The base class logs `task.start` and
   `task.end` with `correlation_id`, `input_hash`, `budget_tokens`,
   `tokens_remaining`, `cost_spent_cents`, and `citations`. Subclasses log
   their own intermediate events.

## Usage

```python
from agents import get_registry, Task, Budget

registry = get_registry()
synth = registry.get("synthesizer")

result = await synth.run(Task(
    question="How will the Affordable Housing Bill affect renters in Nairobi?",
    context={
        "bill_id": "b-2024-14",
        "evidence_chunks": [...],         # from ResearchAgent
        "comparable_cases": [...],        # from HistoricalComparator
        "audit_assumptions": [...],       # from AssumptionAuditor
    },
    budget=Budget(max_tokens=8000, max_cost_cents=Decimal("1.50")),
))
assert result.ai_generated is True
print(result.answer)   # multi-paragraph markdown
print(result.assumptions)  # caveats to surface to the citizen
```

## Tests

```bash
cd services/ai
pytest agents/tests/ -v
```

23 tests cover the registry, budget enforcement, the `run` wrapper contract,
and the architectural invariant that no agent may claim write access to
civic truth.

## Adding a new agent

1. Subclass `BaseAgent` in a new module under `agents/`.
2. Implement `_run(self, task: Task) -> Result`.
3. Declare `permissions` (use `Permission.READ_*` flags; never invent a write flag).
4. Register at startup: `get_registry().register(MyAgent(), capabilities=[...])`.

The registry will reject the registration if the agent claims write access.
