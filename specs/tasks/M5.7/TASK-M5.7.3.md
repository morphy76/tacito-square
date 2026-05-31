# TASK-M5.7.3: Automated Helm Template Testing Suite and NATS Verification

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.7.3                                 |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M5.7                                |
| Depends On    | TASK-M5.7.2                                 |

## Description

Implement the automated Helm validation test script `test/helm/test_agent_standalone_chart.sh`. The script runs dry-run template rendering with custom configuration overlays and asserts formatting correctness, existence of resources, and correct variable serialization. It also documents manual verification procedures using the official `nats` CLI client tool to communicate with the standalone agent.

## Work Items

1. **RED Phase**:
   - Create `test/helm/test_agent_standalone_chart.sh` with basic validation assertions expecting the rendered chart structures to have specific attributes.
   - Run the script and verify that missing charts or missing fields trigger clean exit status failures (RED).

2. **GREEN Phase**:
   - Complete `test/helm/test_agent_standalone_chart.sh` using portable shell assertions (e.g., matching lines/keys of `helm template`).
   - The script must test multiple configurations:
     - Default values.
     - Custom overlays (custom agent name, system prompt, LLM temperature).
     - Secure secret reference inclusion.
   - Implement exit-code based check blocks that fail if any expectation is not met.
   - Provide concrete examples in the chart's `README.md` and script headers detailing the two primary integration patterns for the NATS CLI client:
      - **In-Cluster Interaction**: Running a temporary pod using the official `synadia/nats-box` image:
        `kubectl run nats-box --rm -it --image=synadia/nats-box --namespace=<namespace> -- nats request -s "nats://ts-nats:4222" tacito.agent.<agent-name> "message"`
      - **Local Port-Forwarding**: Port-forwarding the NATS service (`kubectl port-forward svc/ts-nats 4222:4222`) and executing local host commands:
        - `nats sub "tacito.community.>"` to monitor communications.
        - `nats request tacito.agent.<agent-name> '<payload>'` to dispatch queries and verify the agent's replies.
   - Verify all tests pass successfully (GREEN).

3. **REFACTOR Phase**:
   - Clean up code, remove temporary file generation artifacts, and ensure clean output logging in the test script.

## Acceptance Criteria

1. The test script runs completely automated and outputs zero syntax or assertion errors.
2. The verification plan correctly covers the standard `nats` CLI command line syntax.
3. The script is executable (`chmod +x`) and can be integrated into regular CI workflows.
