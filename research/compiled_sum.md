# The compiled Sum function (two args)

00 61 73 6d          ; \0asm (magic)
01 00 00 00          ; version 1

; Type section: 1 type: (i32, i32) -> i32
01 07 01 60 02 7f 7f 01 7f

; Function section: function 0 has type 0
03 02 01 00

; Export section: export "Sum" as func 0
07 07 01 03 53 75 6d 00 00

; Code section: 1 function body
0a 09 01 07 00 20 00 20 01 6a 0f 0b
;              |     |     |  |  +-- 0x0b (end)
;              |     |     |  +----- 0x0f (return)
;              |     |     +-------- 0x6a (i32.add)
;              |     +-------------- 0x20 0x01 (local.get 1)
;              +-------------------- 0x20 0x00 (local.get 0)