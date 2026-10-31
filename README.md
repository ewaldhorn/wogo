# Wogo

**Wogo is an independent, clean-room Go-ish compiler written entirely in Go.** 

*Note: Wogo is an independent open-source project. It is not affiliated with, endorsed by, or sponsored by Google LLC or The Go Authors.*

Basically, Wogo compiles a subset of the Go programming language to Web Assembly, minus nice things like the Go runtime, goroutines, channels and more.

And with subset I really mean a subset. Go is a rich and powerful language, but some of those features, like goroutines, channels and some types of pointers are just a nightmare to port to wasm and quite frankly, I don't need them. This is a hobby project, so I want to use Go for building web things, but not have the entire ecosystem available. Thus, WoGo!

## Seriously use something else for commercial apps

I'd use either [Go](https://go.dev/)'s native wasm target or the insanely cool [TinyGo](https://tinygo.org/) compiler for commercial work. WoGo is highly experimental and so far from complete or done it's not even funny.

## Why man? Why?!

This is the most asked question, and, to be honest, I don't have a clear answer for it. I don't just code to earn a living, I also do it for, well, fun! I've been [tinkering with stuff](https://nofuss.co.za/toys/) forever and really enjoy it as part of my learning process. If I had to guess, I'd say this entire project is just me wanting to really dig deep into understanding WebAssembly better, since I feel like it's a powerful platform that I just don't know enough about. And let's face it, Go's a neat language with a wide range of applications. So being able to leverage that just feels like it will give me a lot of options.

## Dev Log and Docs

I plan to have some [docs](https://ewaldhorn.github.io/wogo/) that I will try to maintain, now and then. No promises, as I often get sucked into coding and that's more fun that figuring out why my frigging html is broken, again.

## Dev Environment

I use Go for development with WasmTime for testing and Wat2Wasm as a reference compiler to sanity check my output.

- [Go](https://go.dev/) (1.27.3)
- [WasmTime](https://wasmtime.dev/) (50.0.1)
- [Wat2Wasm](https://github.com/webassembly/wabt) (1.0.42)

## Acknowledgements & Attribution

This project is an independent, clean-room implementation of the Go programming language specification. 

* **Go Language Specification:** The Go programming language syntax, specifications, and documentation are designed and maintained by the Go Authors and Google LLC. 
* **Official Project:** The official Go language toolchain can be found at [go.dev](https://go.dev).

This project is not affiliated with, endorsed by, or sponsored by Google LLC or the official Go project.
