package analysis

var DefaultDiagnostics = []Diagnostics{
	DiagnosticsJSAssignmentInCondition{},
	DiagnosticsJSDoubleNegation{},
	DiagnosticsJSEqualityCoercion{},
	DiagnosticsJSVar{},
	DiagnosticsJSWith{},
	DiagnosticsQMLAlias{},
	DiagnosticsQMLUnusedImports{},
	DiagnosticsQtQuickLayoutAnchors{},
}
