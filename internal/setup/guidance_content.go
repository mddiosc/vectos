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
		"If you create, move, or edit files while working, prefer an incremental refresh with `changed` paths before retrying search. Use a full reindex only when the affected scope is broad or uncertain.",
		"",
		"Use `grep`, `glob`, and direct file reads only as a fallback when Vectos has no useful results or when you need exact pattern matching.",
		endMarker,
	}, "\n")
}
