package setup

import "strings"

func managedGuidance(startMarker string, endMarker string) string {
	return strings.Join([]string{
		startMarker,
		"## Prefer Vectos MCP",
		"",
		"When Vectos MCP tools are available for a project, prefer the Vectos search tools before using `grep`, `find`, `glob`, or broad file reads.",
		"",
		"For source code and implementation lookups, use `vectos_search_code` when your client prefixes MCP tools with the server name, or `search_code` when it does not.",
		"",
		"For README files, API docs, ADRs, and other documentation, use `vectos_search_docs` or `search_docs` instead of mixing docs into broad file searches.",
		"",
		"If the project is not yet indexed or results are not useful, run `vectos_index_project` or `index_project` and retry. When you need documentation retrieval, index docs separately with `docs: true`.",
		"",
		"Use `grep`, `glob`, and direct file reads only as a fallback when Vectos has no useful results or when you need exact pattern matching.",
		"",
		"## Incremental Reindex (`changed`)",
		"",
		"After creating, moving, or editing files, run `index_project` with the `changed` parameter before retrying search. Use a full reindex (no `changed`) only when the affected scope is broad or uncertain.",
		"",
		"- Pass changed paths as **absolute paths** or as paths **relative to the workspace root**. Do not use paths relative to an individual project or lib root.",
		"- For renames or moves, include **both the old and the new path** in the `changed` list so stale chunks are removed and the new location is indexed.",
		"- After editing documentation files, run a separate `index_project` with `docs: true` to refresh the docs index.",
		"- After changing workspace-level configuration (`nx.json`, `project.json`, lockfiles, `tsconfig.base.json`), perform a **full reindex** — the project graph or file classification may have changed.",
		"",
		"## Nx Monorepo Workflow",
		"",
		"When working inside an Nx monorepo, use the `project` parameter to scope search and indexing to a specific Nx project:",
		"",
		"- `search_code` and `search_docs`: pass `project: \"<project-name>\"` to scope results to that project and its internal dependencies.",
		"- `index_project`: pass `project: \"<project-name>\"` to index a specific Nx project. Vectos automatically indexes the project and all its internal dependency libs.",
		"- `list_projects`: call this tool to discover available Nx project names in the workspace before searching or indexing.",
		"",
		"If the search returns guidance `IDX_MISSING`, index the project first with `index_project` using the correct `project` name.",
		"",
		"If you edit a **shared lib**, its changes are reflected in every project that depends on it. If searches in a downstream project still feel stale after refreshing the lib's project index, refresh the downstream project index as well — or perform a full reindex if the blast radius is unclear.",
		endMarker,
	}, "\n")
}
