# Review Checklist — AgentPipe Multi-Agent Orchestration

## Architecture Decisions
- Verify agent lifecycle boundaries (init, execute, teardown) are explicit in the orchestrator flow. [Ref: architecture-design.md:AgentLifecycle] [Ref: implementation-sketch.go:L96-167]
- Confirm artifact storage is decoupled from orchestration with a stable interface and versioning plan. [Ref: architecture-design.md:ArtifactStorage] [Ref: implementation-sketch.go:L52-93]
- Ensure conversation history captures enough context for downstream agents to act deterministically. [Ref: architecture-design.md:ConversationLog] [Ref: implementation-sketch.go:L33-60]
- Validate dependency passing between tasks is modeled explicitly and respects ordering guarantees. [Ref: architecture-design.md:TaskDependencies] [Ref: implementation-sketch.go:L64-85]
- Check that agent selection strategy is defined (by role, capability, or routing rules). [Ref: architecture-design.md:AgentRouting] [Ref: implementation-sketch.go:L124-150]

## Implementation Approach
- Confirm artifact save path construction is safe and avoids path traversal or invalid filenames. [Ref: architecture-design.md:ArtifactPaths] [Ref: implementation-sketch.go:L76-92]
- Review how mock LLM calls or generated content are stubbed so they can be swapped for real providers. [Ref: architecture-design.md:LLMIntegration] [Ref: implementation-sketch.go:L103-121]
- Ensure error handling paths are consistent and propagate failures to the orchestrator. [Ref: architecture-design.md:ErrorHandling] [Ref: implementation-sketch.go:L142-161]
- Validate that Orchestrator maintains a durable record of turns and supports replayability. [Ref: architecture-design.md:ConversationHistory] [Ref: implementation-sketch.go:L152-160]
- Confirm agent interface is minimal yet sufficient for multi-agent collaboration (task, context, outputs). [Ref: architecture-design.md:AgentInterface] [Ref: implementation-sketch.go:L14-27]

## Risks & Gaps
- Risk: lack of concurrency controls or locking around artifact writes could corrupt outputs under parallel agent runs. [Ref: architecture-design.md:ConcurrencyModel] [Ref: implementation-sketch.go:L70-92]
- Risk: no explicit schema/format for artifact metadata may limit downstream agent reliability. [Ref: architecture-design.md:ArtifactMetadata] [Ref: implementation-sketch.go:L28-41]
- Gap: reviewer/architect agent implementations are TODO, so orchestration path is unvalidated. [Ref: architecture-design.md:AgentImplementations] [Ref: implementation-sketch.go:L130-136]
- Gap: task dependency resolution is manual; consider centralized scheduler or DAG evaluator. [Ref: architecture-design.md:TaskScheduling] [Ref: implementation-sketch.go:L122-150]
- Risk: temporary artifact base dir lacks cleanup policy or retention strategy. [Ref: architecture-design.md:ArtifactRetention] [Ref: implementation-sketch.go:L111-118]
