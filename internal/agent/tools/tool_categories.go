package tools

// ReadOnlyToolNames returns the names of all tools that never modify
// state. This is the canonical source of truth for read-only tool
// classification, used by the auto-mode classifier and the task agent
// tool filter.
func ReadOnlyToolNames() []string {
	return []string{
		ViewToolName,
		LSToolName,
		GlobToolName,
		GrepToolName,
		DiagnosticsToolName,
		ReferencesToolName,
		SymbolsToolName,
		DefinitionToolName,
		CallHierarchyToolName,
		SourcegraphToolName,
		CrushInfoToolName,
		CrushLogsToolName,
		JobOutputToolName,
		TodosToolName,
		QuestionToolName,
		ListMCPResourcesToolName,
		ReadMCPResourceToolName,
		FetchToolName,
		AgenticFetchToolName,
	}
}

// InProjectWriteToolNames returns the names of tools that modify files
// and can be auto-approved when the target path is inside the working
// directory. This is the canonical source of truth for the auto-mode
// classifier's in-project write fast path.
func InProjectWriteToolNames() []string {
	return []string{
		EditToolName,
		MultiEditToolName,
		WriteToolName,
		RenameToolName,
		ReplaceSymbolToolName,
	}
}

// LowRiskToolNames returns the names of tools that are low-risk and
// can be auto-approved without classification. These are tools that
// manage Crush's own state rather than user data.
func LowRiskToolNames() []string {
	return []string{
		JobKillToolName,
		LSPRestartToolName,
	}
}
