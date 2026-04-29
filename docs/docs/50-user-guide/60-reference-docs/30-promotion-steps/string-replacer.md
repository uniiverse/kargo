---
sidebar_label: string-replacer
description: Performs text-based string replacements across YAML manifests using a ConfigMap as the replacement source.
---

# `string-replacer`

`string-replacer` performs text-based string replacements across YAML manifests.
It reads a multi-document YAML file, finds a ConfigMap annotated with
`universe.engineer/string-replacer: "true"`, and uses its `data` entries as
substitution pairs — replacing every occurrence of `REPLACE_ME[KEY]` with the
corresponding value. The step fails if any `REPLACE_ME[...]` placeholders remain
after substitution.

This step is useful when you need to inject environment-specific values into
rendered manifests without forking or modifying upstream Helm charts. It pairs
well with [`kustomize-build`](kustomize-build.md), where a Kustomize
`configMapGenerator` produces the annotated ConfigMap alongside the rendered
output.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `inPath` | `string` | Y | Path to a YAML file containing Kubernetes manifests. The file may contain multiple documents separated by `---`. One of the documents must be a ConfigMap with the annotation `universe.engineer/string-replacer: "true"` whose `data` entries drive the substitutions. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `outPath` | `string` | Y | Path to write the resulting YAML after replacements have been applied. The annotated ConfigMap is preserved in the output. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |

## Behavior

1. The input file is split on YAML document separators (`---`).
2. The step scans for a ConfigMap with the annotation
   `universe.engineer/string-replacer: "true"`. Exactly one such ConfigMap must
   exist — zero or more than one is an error.
3. Each key/value pair in the ConfigMap's `data` field defines a replacement:
   every occurrence of `REPLACE_ME[KEY]` in the entire file is replaced with the
   corresponding value.
4. After all replacements are applied, the step validates that no `REPLACE_ME[...]`
   placeholders remain. If any do, the step fails with an error listing the
   unreplaced placeholders.
5. All documents (including the annotated ConfigMap) are written to `outPath`.

## Examples

### Basic Usage

In this example, a Kustomize build produces a multi-document YAML that includes
an annotated ConfigMap with environment-specific values. The `string-replacer`
step substitutes those values into the rendered manifests.

```yaml
vars:
- name: gitRepo
  value: https://github.com/example/repo.git
steps:
- uses: git-clone
  config:
    repoURL: ${{ vars.gitRepo }}
    checkout:
    - commit: ${{ commitFrom(vars.gitRepo).ID }}
      path: ./src
    - branch: stage/${{ ctx.stage }}
      create: true
      path: ./out
- uses: kustomize-build
  config:
    path: ./src/deploy/${{ ctx.stage }}
    outPath: ./out/manifests.yaml
- uses: string-replacer
  config:
    inPath: ./out/manifests.yaml
    outPath: ./out/manifests.yaml
# Commit, push, etc...
```

Given a `kustomization.yaml` that generates the replacer ConfigMap:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- base.yaml

configMapGenerator:
- name: env-replacements
  options:
    annotations:
      universe.engineer/string-replacer: "true"
  literals:
  - CLUSTER_NAME=prod-us-east1
  - ENVIRONMENT=production
```

And a base manifest containing placeholders:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  config: |
    cluster = "REPLACE_ME[CLUSTER_NAME]"
    env = "REPLACE_ME[ENVIRONMENT]"
```

The output would contain:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  config: |
    cluster = "prod-us-east1"
    env = "production"
```

### In-Place Replacement

The `inPath` and `outPath` may refer to the same file for in-place replacement:

```yaml
- uses: string-replacer
  config:
    inPath: ./out/manifests.yaml
    outPath: ./out/manifests.yaml
```

### Error on Missing Replacements

If a placeholder like `REPLACE_ME[REGION]` appears in the input but `REGION` is
not defined in the replacer ConfigMap's `data`, the step will fail with an error:

```
unreplaced placeholders remain in output: REPLACE_ME[REGION]
```

This ensures that all expected values are provided and prevents deploying
manifests with unresolved placeholders.
