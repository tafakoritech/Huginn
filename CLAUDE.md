## Graphify Code Intelligence

This project has a Graphify knowledge graph under `graphify-out/`.

Use Graphify as the primary codebase navigation and relationship-analysis layer when working on repository-wide questions.

### Rules

- When `graphify-out/graph.json` exists, use Graphify before broad repository searches for:
  - architecture questions
  - dependency analysis
  - cross-file relationships
  - execution-flow tracing
  - impact analysis
  - debugging involving multiple components
  - locating related implementations

- Use:
  - `graphify query "<question>"` for scoped codebase questions
  - `graphify path "<A>" "<B>"` to investigate relationships and paths between components
  - `graphify explain "<concept>"` for focused concepts or components

- Prefer the scoped output from `query`, `path`, and `explain` over reading the entire `GRAPH_REPORT.md` or performing broad `grep` searches.

- If `graphify-out/wiki/index.md` exists, use it for high-level project navigation before browsing the repository broadly.

- Read `graphify-out/GRAPH_REPORT.md` for:
  - broad architecture reviews
  - understanding project communities
  - high-level system structure
  - cases where `query`, `path`, or `explain` do not provide sufficient context

- Graphify is a navigation and analysis aid, not the source of truth.
  Always inspect the actual source code before making implementation decisions or modifying code.

- If Graphify and the source code disagree, the actual source code is authoritative.

- Do not assume that a node, edge, community, or relationship in the graph means that functionality is currently active or reachable.

### After Code Changes

After modifying code, update the Graphify graph when the changes affect code structure, dependencies, APIs, modules, classes, functions, or cross-file relationships:

graphify update . is AST-only and does not require an LLM API call.

Do not regenerate the entire semantic graph unnecessarily after every small change.

Important

Do not blindly run Graphify for every request.

Use Graphify when it provides useful structural or relationship context. For simple questions about a known file, function, or small change, inspect the relevant source directly.
