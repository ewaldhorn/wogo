# WoGo

A small compiler that takes a subset of the [Go](https://go.dev/) programming language and compiles it to Web Assembly.

And with subset I really mean a subset. Go is a rich and powerful language, but some of those features, like goroutines, channels and some types of pointers are just a nightmare to port to wasm and quite frankly, I don't need them. This is a hobby project, so I want to use Go for building web things, but not have the entire ecosystem available. Thus, WoGo!

## Seriously use something else for commercial apps

I'd use either Go's native wasm target or the insanely cool [TinyGo](https://tinygo.org/) compiler for commercial work. WoGo is highly experimental and so far from complete or done it's not even funny.
