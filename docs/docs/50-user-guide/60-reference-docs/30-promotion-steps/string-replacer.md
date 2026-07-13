---
sidebar_label: string-replacer
description: Performs text-based string replacements across YAML manifests using a ConfigMap as the replacement source.
---

# `string-replacer`

`string-replacer` performs text-based string replacements across YAML manifests.
It reads a multi-document YAML file and collects substitution pairs from two
optional sources — a ConfigMap annotated with
`universe.engineer/string-replacer: "true"` (its `data` entries) and an inline
`replacements` map on the step config — then replaces every occurrence of
`REPLACE_ME[KEY]` with the corresponding value. The step fails if any
`REPLACE_ME[...]` placeholders remain after substitution.

The inline `replacements` map exists for values that aren't known until
promotion time and therefore can't be baked into a checked-in `configMapGenerator`
— most commonly the freight commit SHA, injected via an expression such as
`${{ commitFrom(vars.repoURL).ID[:7] }}`.

This step is useful when you need to inject environment-specific values into
rendered manifests without forking or modifying upstream Helm charts. It pairs
well with [`kustomize-build`](kustomize-build.md), where a Kustomize
`configMapGenerator` produces the annotated ConfigMap alongside the rendered
output.

## Configuration

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `inPath` | `string` | Y | Path to a YAML file containing Kubernetes manifests. The file may contain multiple documents separated by `---`. One of the documents may be a ConfigMap with the annotation `universe.engineer/string-replacer: "true"` whose `data` entries drive the substitutions. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `outPath` | `string` | Y | Path to write the resulting YAML after replacements have been applied. The annotated ConfigMap is preserved in the output. This path is relative to the temporary workspace that Kargo provisions for use by the promotion process. |
| `replacements` | `map[string]string` | N | Inline `KEY: value` pairs applied as `REPLACE_ME[KEY]` → `value` substitutions. Merged with the annotated ConfigMap's `data`; on a key collision the inline value wins. Use this for values only known at promotion time (e.g. the freight commit SHA). |

## Behavior

1. The input file is split on YAML document separators (`---`).
2. The step scans for a ConfigMap with the annotation
   `universe.engineer/string-replacer: "true"`. At most one such ConfigMap may
   exist — more than one is an error.
3. The ConfigMap's `data` entries (if any) are merged with the inline
   `replacements` map. On a key collision the inline value wins. If neither
   source provides any entries, the step skips replacement and proceeds directly
   to placeholder validation (step 5).
4. Each merged key/value pair defines a replacement: every occurrence of
   `REPLACE_ME[KEY]` in the entire file is replaced with the corresponding value.
5. After all replacements are applied, the step validates that no `REPLACE_ME[...]`
   placeholders remain. If any do, the step fails with an error listing the
   unreplaced placeholders.
6. All documents (including the annotated ConfigMap) are written to `outPath`.

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

### Injecting a Promotion-Time Value

Some values aren't known until promotion runs and can't be baked into a
checked-in `configMapGenerator` — the freight commit SHA is the canonical
example. Supply those through the inline `replacements` map, populated from a
promotion expression:

```yaml
- uses: string-replacer
  config:
    inPath: ./out/manifests.yaml
    outPath: ./out/manifests.yaml
    replacements:
      FREIGHT_SHA: ${{ commitFrom(vars.repoURL).ID[:7] }}
```

A base manifest can then reference it like any other placeholder:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    metadata:
      labels:
        app.kubernetes.io/version: REPLACE_ME[FREIGHT_SHA]
```

Inline replacements are merged with the annotated ConfigMap's `data`, so this
composes with the environment-specific values in the [Basic Usage](#basic-usage)
example. On a key collision the inline value wins.

### Error on Missing Replacements

If a placeholder like `REPLACE_ME[REGION]` appears in the input but `REGION` is
not defined in the replacer ConfigMap's `data`, the step will fail with an error:

```
unreplaced placeholders remain in output: REPLACE_ME[REGION]
```

This ensures that all expected values are provided and prevents deploying
manifests with unresolved placeholders.
