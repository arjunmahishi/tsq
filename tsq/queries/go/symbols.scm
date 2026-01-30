; Function declarations
(function_declaration
  name: (identifier) @name
  parameters: (parameter_list) @params
  result: (_)? @result) @function

; Method declarations
(method_declaration
  receiver: (parameter_list) @receiver
  name: (field_identifier) @name
  parameters: (parameter_list) @params
  result: (_)? @result) @method

; Type declarations (struct, interface, type alias)
(type_declaration
  (type_spec
    name: (type_identifier) @name
    type: (_) @type_def)) @type

; Const declarations
(const_declaration
  (const_spec
    name: (identifier) @name
    type: (_)? @type
    value: (_)? @value)) @const

; Var declarations
(var_declaration
  (var_spec
    name: (identifier) @name
    type: (_)? @type
    value: (_)? @value)) @var
