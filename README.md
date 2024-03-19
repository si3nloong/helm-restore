# helm-restore

> A CLI to restore your helm charts. As long as your kubernetes cluster is available, you can restore the helm chart using `helm-restore` CLI.

## ❓ Why `helm-restore`?

Encountering the scenarios mentioned below? It's time to leverage the power of `helm-restore` CLI.

- Your team has lost the original helm charts.
- Previous developer has left the organization without providing the original Helm charts.
- Third party has assisted in setting up components, leaving you without the original Helm charts.

Having faced the same challenges, I developed this tool to simplify the process of recovering Helm charts.

## 🔨 Installation

### Brew

```console
brew install helm-restore
```

### Go

```console
go install github.com/si3nloong/helm-restore@main
```

### Distribution

[Downloads](https://github.com/si3nloong/helm-restore/releases/tag/v1.0.0)

## 🥢 How to use?

```bash
helm-restore --latest -o dist
```

This will take some time if you have many charts. After it complete, you will see your charts inside `dist` folder.

## ⚙️ Command line tool

### Syntax

Use the following syntax to run `helm-restore` commands from your terminal window:

```bash
helm-restore [command] [flags]
```

where `command`, and `flags` are:

- `command`: Specifies the operation that you want to perform.
- `flags`: Specifies optional flags.

### Cheat Sheet

| Flags                | Description                                                                                       |
| -------------------- | ------------------------------------------------------------------------------------------------- |
| --latest             | Only download the latest chart                                                                    |
| -f <kubeconfig_file> | Load the cluster using the specific kubeconfig file instead of using default `$HOME/.kube/config` |
| -o <output_folder>   | Store the helm charts in the specific folder                                                      |
| -context <context>   | Specify the kubernetes context                                                                    |

**Examples :**

```bash
helm-restore --latest # download the latest charts only
helm-restore -o dist # download the charts and store in `dist` folder
```

## 📄 License

[MIT](https://github.com/si3nloong/helm-restore/blob/main/LICENSE)

Copyright (c) 2024-present, SianLoong Lee
