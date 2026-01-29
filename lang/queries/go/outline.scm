; Package declaration
(package_clause
  (package_identifier) @package)

; Import declarations
(import_declaration
  (import_spec
    name: (package_identifier)? @alias
    path: (interpreted_string_literal) @path)) @import

(import_declaration
  (import_spec_list
    (import_spec
      name: (package_identifier)? @alias
      path: (interpreted_string_literal) @path))) @import_list

; Function declarations
(function_declaration
  name: (identifier) @func_name) @function

; Method declarations
(method_declaration
  receiver: (parameter_list
    (parameter_declaration
      type: (_) @receiver_type))
  name: (field_identifier) @method_name) @method

; Type declarations - structs
(type_declaration
  (type_spec
    name: (type_identifier) @type_name
    type: (struct_type) @struct_body)) @struct

; Type declarations - interfaces
(type_declaration
  (type_spec
    name: (type_identifier) @type_name
    type: (interface_type) @interface_body)) @interface

; Type declarations - type aliases and other types (not struct/interface)
(type_declaration
  (type_spec
    name: (type_identifier) @type_name
    type: (type_identifier) @aliased_type)) @type_alias

(type_declaration
  (type_spec
    name: (type_identifier) @type_name
    type: (pointer_type) @ptr_type)) @type_ptr

(type_declaration
  (type_spec
    name: (type_identifier) @type_name
    type: (slice_type) @slice_type_body)) @type_slice

(type_declaration
  (type_spec
    name: (type_identifier) @type_name
    type: (map_type) @map_type_body)) @type_map

(type_declaration
  (type_spec
    name: (type_identifier) @type_name
    type: (function_type) @func_type_body)) @type_func

; Const declarations
(const_declaration
  (const_spec
    name: (identifier) @const_name)) @const

; Var declarations  
(var_declaration
  (var_spec
    name: (identifier) @var_name)) @var
