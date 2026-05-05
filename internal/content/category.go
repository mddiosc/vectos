package content

import "strings"

func ClassifyCategory(language string) string {
	switch {
	case language == "shell":
		return "scripts"
	case language == "markdown", language == "gitignore", language == "rst", language == "asciidoc", language == "latex", language == "text":
		return "docs"
	case language == "json", language == "toml", language == "ini", language == "xml", language == "properties", language == "makefile", language == "gradle", language == "lockfile", language == "config":
		return classifyMetadataCategory(language)
	case language == "dockerfile", strings.HasPrefix(language, "yaml"), strings.HasPrefix(language, "bazel"):
		return "infra_config"
	default:
		return "source"
	}
}

func classifyMetadataCategory(language string) string {
	switch language {
	case "json", "toml", "properties", "xml", "makefile", "gradle", "lockfile":
		return "dependency_metadata"
	default:
		return "infra_config"
	}
}
