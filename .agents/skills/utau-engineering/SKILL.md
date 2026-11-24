---
name: utau-engineering
description: Research, design, implement, and test a modern UTAU-compatible singing voice synthesizer in Go. Use when working on synthesis, classic UTAU compatibility, voicebanks, oto.ini, note timing, pitch, resamplers, wavtool, phonemization, project formats, OpenUtau reference code, the general UTAU architecture, or historical Japanese UTAU documentation.
disable-model-invocation: false
---

# UTAU Engineering

Help develop a modern UTAU-compatible singing voice synthesizer in Go, using classic UTAU as the primary compatibility target and OpenUtau's C# implementation as a secondary reference.

The long-term goal is a maintainable, idiomatic Go synthesizer that preserves classic UTAU behavior where compatibility requires it, while allowing modern internal architecture and future improvements.

The current priority is the synthesis engine. CLI and GUI development are future phases; do not divert current work toward them unless explicitly requested or necessary for an engine interface decision.

## Core principles

1. **Compatibility is a tested behavioral contract.** Do not assume that matching OpenUtau automatically means matching classic UTAU.
2. **Modern internals, compatible behavior.** Prefer clear Go abstractions and testable components without silently changing externally observable legacy behavior.
3. **Evidence before implementation.** Inspect the existing project, relevant source code, documentation, and tests before proposing significant changes.
4. **Preserve uncertainty.** Distinguish verified behavior from inferred behavior, approximations, and unresolved questions.
5. **Regression tests are permanent knowledge.** When a behavior or compatibility quirk is established, encode it in a test wherever practical.
6. **Respect project conventions.** Follow the existing repository's architecture, naming, error handling, dependencies, and formatting unless there is a concrete reason to change them.
7. **Do not invent specifications.** If the evidence is incomplete, state what is unknown and propose the smallest experiment that can resolve it.

## Initial workflow

Before modifying code:

1. Inspect the repository structure, `go.mod`, package boundaries, existing tests, and relevant documentation.
2. Identify the current implementation stage and the specific component being changed.
3. Trace the relevant data flow from its input through processing to its output.
4. Search for existing behavior, tests, constants, and helper functions before introducing alternatives.
5. Identify compatibility requirements and any uncertainty that could affect the implementation.
6. Propose a focused change rather than an unrelated redesign.

Do not assume files, packages, functions, or architecture exist without inspecting the repository.

Avoid reimplementing functionality that already exists unless the current implementation is incorrect, incomplete, or unsuitable for the required behavior.

## Evidence and source hierarchy

Use sources according to the question being investigated. No single source is authoritative for every kind of behavior.

### Historical compatibility

Prioritize:

1. Reproducible observations from classic UTAU or its original tools, with the environment and inputs documented.
2. Contemporary documentation, original distribution files, format specifications, and historical Japanese community sources.
3. Archived versions of relevant documentation or websites.
4. OpenUtau source code as evidence of a particular implementation's behavior.
5. Later explanations, recollections, and secondary sources.
6. Inference from naming, conventions, or incomplete examples.

### Current OpenUtau behavior

Prioritize the actual checked-out source revision, relevant tests, and executable behavior. Record the repository revision when findings depend on implementation details.

### General engineering decisions

Prefer existing project conventions, standard Go practices, measured performance, and explicit requirements. Historical quirks should not dictate internal architecture when compatibility can be preserved at a boundary.

When sources conflict, explain the conflict. Do not silently choose whichever source is easiest to implement.

## Researching Japanese documentation

Search in both Japanese and English. Japanese terminology is often essential when researching classic UTAU behavior.

Use combinations of terms appropriate to the question, including:

- `UTAU 仕様`
- `UTAU 使い方`
- `UTAU 音源`
- `UTAU 原音設定`
- `UTAU oto.ini`
- `UTAU UST`
- `UTAU エンジン`
- `UTAU resampler`
- `UTAU wavtool`
- `UTAU ピッチ`
- `UTAU 音符 長さ`
- `UTAU エイリアス`
- `UTAU 互換`
- `UTAU 不具合`
- `UTAU 配布終了`
- `UTAU 配布停止`

These are starting points, not a fixed search vocabulary. Expand them with terminology found in the source material.

For each useful source:

- Preserve the original Japanese term and provide a contextual English translation.
- Record the original URL, title, author if known, and publication or capture date when available.
- Quote only the passage needed to support the finding.
- Distinguish an author's observation from a documented specification.
- Check whether the page describes classic UTAU, OpenUtau, a particular plugin, or another implementation.
- Search for archived copies if the original page is missing.
- Do not interpret an inaccessible page or deleted URL as proof of any particular behavior.

If a technical Japanese term has multiple plausible meanings, retain the original term until its meaning is established.

## Using OpenUtau as a reference implementation

Use OpenUtau's C# code to discover algorithms, data structures, compatibility handling, and edge cases.

When investigating a feature:

1. Search for the relevant type, method, property, constant, and related tests.
2. Trace callers and callees far enough to understand the actual behavior.
3. Identify the source data, intermediate representations, defaults, and final output.
4. Inspect relevant history or revision information when behavior may have changed.
5. Determine whether the behavior is intended for classic compatibility, an OpenUtau-specific feature, or a modern design choice.
6. Compare the findings with historical evidence when classic UTAU compatibility matters.
7. Translate the behavior into explicit requirements and Go tests before copying complex logic.

Record repository-relative paths, symbol names, and relevant line ranges when available. Include a commit or revision identifier when the claim depends on a particular version.

Do not assume that:

- A class name describes the complete algorithm.
- A modern implementation preserves every legacy quirk.
- Similar file formats imply identical semantics.
- A feature implemented in OpenUtau existed in classic UTAU.
- A difference is a bug merely because two implementations disagree.

Reproduce behavior when evidence justifies it, not simply because a reference implementation happens to do it.

If present, you may reference the OpenUtau source code from a local clone of the repository (`OpenUtau/`, `_OpenUtau/`, `.OpenUtau/`, etc.) rather than looking up the online repository. If not present, search the OpenUtau source code online.

## Go implementation principles

Prefer idiomatic, maintainable Go.

- Keep synthesis algorithms separate from file parsing, filesystem access, command-line concerns, and GUI concerns.
- Use small interfaces when they enable meaningful substitution or testing; avoid abstractions that exist only for hypothetical future needs.
- Represent important domain concepts explicitly when doing so prevents unit errors or ambiguous state.
- Keep timing units, sample counts, rates, pitch representations, and coordinate systems unambiguous.
- Avoid unnecessary global mutable state.
- Propagate errors with useful context instead of silently swallowing failures.
- Validate untrusted file input and guard against invalid values, integer overflow, and unreasonable allocations.
- Use deterministic tests for mathematical and signal-processing behavior whenever possible.
- Profile before optimizing. Use benchmarks to verify performance improvements.
- Avoid introducing dependencies without a clear benefit.

Do not translate C# into Go mechanically. Preserve the relevant semantics while using appropriate Go data structures and control flow.

## Synthesis engine investigation

For each engine subsystem, determine its responsibilities, inputs, outputs, units, state, error conditions, and compatibility requirements before changing its design.

Relevant areas may include:

- Voicebank discovery and voicebank metadata.
- `oto.ini` parsing and alias resolution.
- Phoneme and lyric interpretation.
- Note timing and tempo handling.
- Pitch curves and pitch-bend interpolation.
- Sample selection, offset, consonant, cutoff, preutterance, and overlap semantics.
- Resampler invocation and parameter interpretation.
- Wavtool behavior and rendered-fragment assembly.
- Sample-rate conversion, audio buffering, and output format.
- Flags, envelopes, modulation, and other engine parameters.
- Legacy configuration, text encodings, path conventions, and malformed input handling.

This is an investigation checklist, not a claim that every subsystem must be implemented immediately.

For every feature, distinguish:

1. The external format or API contract.
2. The legacy behavior that must be reproduced.
3. The internal Go representation.
4. The algorithm or processing stage.
5. The observable output.
6. The tests that demonstrate correctness.

Avoid conflating note scheduling, phonemization, voicebank lookup, waveform generation, and audio assembly merely because classic tools pass data between them.

## Compatibility model

Treat compatibility as a collection of explicit, independently testable requirements.

For each relevant behavior, classify the current evidence as one of:

- **Verified legacy behavior:** demonstrated against classic UTAU or supported by strong historical evidence.
- **Documented legacy behavior:** described by a credible source but not independently reproduced.
- **Reference behavior:** observed in OpenUtau or another implementation without proof that it matches classic UTAU.
- **Modern design choice:** intentionally different internal or external behavior.
- **Unresolved:** insufficient evidence to decide.

Use these classifications in research notes, not as a substitute for tests.

Where necessary, isolate legacy compatibility behavior behind small, well-defined functions or adapters. Do not scatter unexplained compatibility conditionals throughout unrelated algorithms.

When a modern behavior conflicts with legacy compatibility, prefer a design that preserves the classic contract without unnecessarily constraining internal implementation. If that is not possible, explain the trade-off explicitly.

## Testing requirements

Choose tests based on the type of behavior being implemented.

### Unit tests

Use for parsing, numeric transformations, interpolation, alias resolution, parameter interpretation, boundary conditions, and other deterministic logic.

### Regression tests

Add a test whenever a discovered bug, legacy quirk, or compatibility requirement is likely to recur.

Include the origin of the expected behavior in a test comment or linked research note when it is not self-evident.

### Golden tests

Where appropriate, compare generated intermediate data, audio buffers, rendered fragments, or other deterministic output against a known reference.

Document the provenance of the expected result and any permitted tolerance. Do not invent expected values merely to make the test pass.

### Integration tests

Test interactions between the relevant engine components, including file input, voicebank lookup, parameter propagation, rendering, and audio assembly as the implementation supports them.

### Differential tests

When classic UTAU or another reference implementation can be executed, use identical inputs and compare outputs or observable behavior.

Before treating a mismatch as a defect, account for differences in versions, external resamplers, sample rates, floating-point precision, environment, and undocumented behavior.

### Fuzzing and robustness

For parsers and complex input boundaries, consider fuzz tests, malformed-input tests, and resource-limit tests.

Run relevant tests after a change. Do not claim tests passed unless they were actually run and their results observed.

## Response format for engineering work

For nontrivial changes, structure the explanation around:

1. **Finding:** What behavior or problem was identified?
2. **Evidence:** What code, documentation, experiment, or test supports it?
3. **Decision:** What should the Go implementation do, and why?
4. **Change:** Which components should change?
5. **Verification:** Which tests or experiments establish correctness?
6. **Uncertainty:** What remains unverified?

For simple, well-understood tasks, respond concisely instead of mechanically filling every heading.

Before making a large change, explain material compatibility risks and architectural trade-offs. Prefer focused, reviewable changes over broad refactors.

## Definition of success

The skill is successful when it helps the project:

- Make progress on the engine without repeatedly rediscovering the same facts.
- Trace implementation decisions back to inspectable evidence.
- Reproduce classic UTAU behavior deliberately rather than accidentally.
- Use OpenUtau effectively without treating it as an infallible specification.
- Turn discoveries into durable documentation and regression tests.
- Keep the Go implementation modular, testable, and ready for future interfaces.
- Make uncertainty visible before it becomes an undocumented compatibility bug.
