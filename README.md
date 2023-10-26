# kargo

`kargo` is a command-line application to deploy your Kubernetes application directly and indirectly.

By "direct", it means `kargo` is able to deploy the app by directly calling popular commands like `kubectl`, `kustomize`, `helm`, `kompose`.

By "indirect", it means `kargo` can let ArgoCD deploy your application using kustomize/helm/kompose by setting up ArgoCD on behalf of you.

Everyone in your team is able to trigger deployments via `kargo`, no matter which Kubernetes deployment tool you or your teammate is using. That's the benefit of introducing `kargo` into your environment.

If you're a part of the platform team, you'll probably want to
encourage or enforce use of `kargo`, so that you can standardize
deployments without forcing your team to use kubectl/kustomize/helm/argocd.

## Usage

### Standalone

`kargo` has two commands, `kargo plan` and `kargo apply`.

`plan` outputs the diff between the current state and the desired state of your application deployment, so that you can review changes before they are applied.

`apply` runs the deployment`

### Embedded

`kargo` can be embedded into your own Go application.

You instantiate a `kargo.Config` and a `kargo.Generator`, and let the generator generates the commands to be executed to either "plan" or "apply" the config changes you made.


```go
import (
  "github.com/mumoshu/kargo"
)

func yourAppDeploymentTool() error {
  c := &kargo.Config{
    Name:    "myapp",
    Path:    "testdata/compose",
    Kompose: &kargo.Kompose{},
    ArgoCD:  &kargo.ArgoCD{},
  }

  g := &kargo.Generator{
    GetValue: func(key string) (string, error) {
      return yourSecretManager.Get(key)
    },
    TailLogs: false,
  }

  cmds, err := g.ExecCmds(c, targ)
  if err != nil {
    return err
  }

  // Run the cmds with your favorite command runner.
}
```

See [generator.go](./generator.go) and `generator_*_test.go` files for more information.

## Configuration

The below is the reference configuration that covers all the required and optional fields available for this provider:

```yaml
# This maps to --plugin-env in case you're going to uses the `argocd` option below.
# Otherwise all the envs are set before calling commands (like kompose, kustomize, kubectl, helm, etc.)
env:
- name: STAGE
  value: prod
- name: FOO
  valueFrom: component_name.foo
# kustomize instructs kargo to deploy the app using `kustomize`.
# It has two major modes. The first mode directly calls `kustomize`, whereas
# the second indirectly call it via `argocd`.
# The first mode is triggered by setting only `helm`.
# The second is enabled when you set `argocd` along with `kustomize`.
kustomize:
  # kustomize.image maps to --kustomize-image of argocd-app-create.
  image:
# helm instructs kargo to deploy the app using `helm`.
# It has two major modes. The first mode directly calls `helm`, whereas
# the second indirectly call it via `argocd`.
# The first mode is triggered by setting only `helm`.
# The second is enabled when you set `argocd` along with `helm`.
helm:
  # helm.repo maps to --repo of argocd-app-create
  # in case kubernetes.argocd is not empty.
  repo: https://charts.helm.sh/stable
  # --helm-chart
  chart: mychart
  # --revision
  version: 1.2.3
  # helm.set corresponds to `--helm-set $name=$value` flags of `argocd app create` command
  set:
  - name: foo
    value: foo
  - name: bar
    valueFrom: component_name.bar
  valuesFiles:
  - path/to/values.yaml
argocd:
  # argocd.repo maps to --repo of argocd-app-create.
  repo: github.com/myorg/myrepo.git
  # argocd.path maps to --path of argocd-app-create.
  # Note: In case you had kubernetes.dir along with argocd.path,
  # kargo automatically git-push the content of dir to $argocd_repo/$argocd_path.
  # To opt-out of it, set `push: false`.
  path: path/to/dir/in/repo
  # --dir-recurse
  dirRecurse: true
  # --dest-namespace
  namespace: default
  # serverFrom maps to --dest-server where the flag value is take from the output of another kargo component
  serverFrom: component_name.k8s_endpoint
  # Note that the config management plugin definition in the configmap
  # and the --config-management-plugin flag passed to argocd-app-create # command is auto-generated.
```

## Deploying to multiple environments

`kargo` does not have a "environments" concept or any feature related to that.
It's intentionally out of the scope of this project to keep it simple.

However, you can still support multiple environments just by creating one `kargo.yaml` per environment.

Let's suppose you are deploying to two environments, `production` and `preview`.
You start by creating `production.kargo.yaml` and `preview.kargo.yaml`.

When you want `kargo` to deploy to a specific environment, just give the corresponding `kargo` config file via the `-f` flag.

That is, you'll run `kargo` like `kargo -f production.kargo.yaml apply` for a production deployment, whereas it would be `kargo -f preview.kargo.yaml apply` for a preview deployment.

You'll ask how one could reduce the duplications and boilerplates in the two config files.

`kargo` assumes you would like to use a tool like [cue](https://cuelang.org/) or [jsonnnet](https://jsonnet.org/) to produce the `kargo` config files. That way, you can use advanded features provided in those tools to reduce boiler plates and introduce any abstractions to compose your `kargo` config files in a maintainable manner.
