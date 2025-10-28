# hpademo

`hpademo` is a simple demo for Kubernetes Horizontal Pod Autoscaler (HPA), written in Go and compiled to WebAssembly (WASM) in order to run in a web browser.

Online demo: https://udhos.github.io/hpademo/www/

# TODO

- [X] scale up limit
- [ ] scale down 5min anti-flap

# clone

```bash
git clone https://github.com/udhos/hpademo
cd hpademo
```

# test

```bash
./test.sh
```

# build

```bash
./build.sh
```

# run

```bash
./run-serve-www.sh
```
