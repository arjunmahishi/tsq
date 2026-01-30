; Function/method calls
(call_expression
  function: (identifier) @call)

(call_expression
  function: (selector_expression
    operand: (_) @receiver
    field: (field_identifier) @call))

; Type references
(type_identifier) @type_ref

; Identifiers (variable references)
(identifier) @ident

; Selector expressions (field access, method calls)
(selector_expression
  operand: (_) @operand
  field: (field_identifier) @field)

; Composite literal types
(composite_literal
  type: (type_identifier) @composite_type)

; Short var declarations
(short_var_declaration
  left: (expression_list
    (identifier) @short_var))
