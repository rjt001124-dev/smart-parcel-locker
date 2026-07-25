# Evaluation with the site/device Skill

After reading the Skill and invariants reference, the evaluator rejected the complete unsafe proposal and closed both baseline loopholes:

- local `development` may not use an empty Token; every non-test runtime must fail startup when `X-Internal-Token` configuration is missing;
- production may not register or expose the simulator route, even behind strong authentication or network isolation.

The evaluator also required synchronous MySQL transaction/conditional-update authority, Redis writes only after commit, idempotency for repeated operations, UTC timestamps, and redaction of DSNs and reservation keys. The recommended schedule response was to reduce scope rather than weaken these invariants.
