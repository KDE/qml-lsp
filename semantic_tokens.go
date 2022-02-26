package main

type NumSemanticTokenKind uint32
type NumSemanticTokenModifier uint32

const (
	SemanticTokenTypeNamespace     = "namespace"
	SemanticTokenTypeType          = "type"
	SemanticTokenTypeClass         = "class"
	SemanticTokenTypeEnum          = "enum"
	SemanticTokenTypeInterface     = "interface"
	SemanticTokenTypeStruct        = "struct"
	SemanticTokenTypeTypeParameter = "typeParameter"
	SemanticTokenTypeParameter     = "parameter"
	SemanticTokenTypeVariable      = "variable"
	SemanticTokenTypeProperty      = "property"
	SemanticTokenTypeEnumMember    = "enumMember"
	SemanticTokenTypeEvent         = "event"
	SemanticTokenTypeFunction      = "function"
	SemanticTokenTypeMethod        = "method"
	SemanticTokenTypeMacro         = "macro"
	SemanticTokenTypeKeyword       = "keyword"
	SemanticTokenTypeModifier      = "modifier"
	SemanticTokenTypeComment       = "comment"
	SemanticTokenTypeString        = "string"
	SemanticTokenTypeNumber        = "number"
	SemanticTokenTypeRegexp        = "regexp"
	SemanticTokenTypeOperator      = "operator"
)

const (
	NumSemanticTokenTypeNamespace NumSemanticTokenKind = iota
	NumSemanticTokenTypeType
	NumSemanticTokenTypeClass
	NumSemanticTokenTypeEnum
	NumSemanticTokenTypeInterface
	NumSemanticTokenTypeStruct
	NumSemanticTokenTypeTypeParameter
	NumSemanticTokenTypeParameter
	NumSemanticTokenTypeVariable
	NumSemanticTokenTypeProperty
	NumSemanticTokenTypeEnumMember
	NumSemanticTokenTypeEvent
	NumSemanticTokenTypeFunction
	NumSemanticTokenTypeMethod
	NumSemanticTokenTypeMacro
	NumSemanticTokenTypeKeyword
	NumSemanticTokenTypeModifier
	NumSemanticTokenTypeComment
	NumSemanticTokenTypeString
	NumSemanticTokenTypeNumber
	NumSemanticTokenTypeRegexp
	NumSemanticTokenTypeOperator
)

const (
	SemanticTokenTypeDeclaration    = "declaration"
	SemanticTokenTypeDefinition     = "definition"
	SemanticTokenTypeReadonly       = "readonly"
	SemanticTokenTypeStatic         = "static"
	SemanticTokenTypeDeprecated     = "deprecated"
	SemanticTokenTypeAbstract       = "abstract"
	SemanticTokenTypeAsync          = "async"
	SemanticTokenTypeModification   = "modification"
	SemanticTokenTypeDocumentation  = "documentation"
	SemanticTokenTypeDefaultLibrary = "defaultLibrary"
)

const (
	NumSemanticTokenTypeDeclaration NumSemanticTokenModifier = 1 << iota
	NumSemanticTokenTypeDefinition
	NumSemanticTokenTypeReadonly
	NumSemanticTokenTypeStatic
	NumSemanticTokenTypeDeprecated
	NumSemanticTokenTypeAbstract
	NumSemanticTokenTypeAsync
	NumSemanticTokenTypeModification
	NumSemanticTokenTypeDocumentation
	NumSemanticTokenTypeDefaultLibrary
)
