# I need to support the logical NOT operator

For example, !true == false. Since I use i32 (0,1) to represent true and false, it looks like I can
use the [eqz](https://github.com/sunfishcode/wasm-reference-manual/blob/master/WebAssembly.md#integer-equal-to-zero) operator to get this done.

According to the docs, this should work. Gonna try it out.
