# Envoy filters

## Prerequisites
- Install Go
- Install [Tinygo](https://tinygo.org/getting-started/install/macos/)

## Build

Example:
```
make build target=headers
```

wasm filter will be generated under the `filters/<filter_name>/` as `main.wasm`

Include the wasm filter in the `envoy.yaml` and examples are shown in the filter's sub-directory.
