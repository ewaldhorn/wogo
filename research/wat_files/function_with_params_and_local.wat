(func $DoubleAndAdd (param $a i32) (result i32)
  (local $temp i32)
  local.get $a
  i32.const 2
  i32.mul
  local.set $temp
  local.get $temp
  i32.const 5
  i32.add
  return
)

;; everything shares index space
;; so the param is index 0
;; and the local is index 1