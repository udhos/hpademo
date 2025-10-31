[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/udhos/hpademo/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/udhos/hpademo)](https://goreportcard.com/report/github.com/udhos/hpademo)
[![Go Reference](https://pkg.go.dev/badge/github.com/udhos/hpademo.svg)](https://pkg.go.dev/github.com/udhos/hpademo)

# hpademo

`hpademo` is a simple demo for Kubernetes Horizontal Pod Autoscaler (HPA), written in Go and compiled to WebAssembly in order to run in a web browser.

Online demo: https://udhos.github.io/hpademo/www/

# features

- Simulate HPA based on CPU.
- Chart for number of replicas.
- Chart for per-pod CPU usage.
- Chart for total unmet CPU load.
- Dark/light modes.
- Customizable:
  - Inject total CPU usage.
  - POD CPU request.
  - POD CPU limit.
  - HPA min replicas.
  - HPA max replicas.
  - HPA targe cpu utilization percentage.
  - Chart data history size (300s default).
  - Scale down stabilization window (300s default).
- Non-customizable:
  - 10% HPA Scale Tolerance.
  - 15s HPA Scale Period.

# clone

```bash
git clone https://github.com/udhos/hpademo
cd hpademo
```

# edit

Set `GOOS=js GOARCH=wasm` before starting your editor/IDE.

```bash
export GOOS=js GOARCH=wasm
code .
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

Then open your web browser at http://localhost:8080
