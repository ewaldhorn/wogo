(module
  (func $my_function (result i32)
    (local $my_var i32)

    i32.const 10
    local.set $my_var

    local.get $my_var
  )
)
