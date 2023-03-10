# kargo

`kargo` is a command-line application to deploy your Kubernetes application in various ways directly or indirectly.

By "direct", it means `kargo` is able to deploy the app by directly calling popular commands like `kubectl`, `kustomize`, `helm`, `kompose`.

By "indirect", it means `kargo` can let ArgoCD deploy your application using kustomize/helm/kompose by setting up ArgoCD on behalf of you.
Further more, it is able to generate a continuous deployment workflow for CodeBuild and GitHub Actions, so that the target system can deploy your application via
the CI/CD system of your choice.

The below is the reference configuration that covers all the required and optional fields available for this provider:

```yaml
# This maps to --plugin-env in case you're going to uses the `argocd` option below.
# Otherwise all the envs are set before calling commands (like kompose, kustomize, kubectl, helm, etc.)
env:
- name: STAGE
  value: prod
- name: FOO
  valueFrom: component_name.foo
# kustomize instructs kanvas to deploy the app using `kustomize`.
# It has two major modes. The first mode directly calls `kustomize`, whereas
# the second indirectly call it via `argocd`.
# The first mode is triggered by setting only `helm`.
# The second is enabled when you set `argocd` along with `kustomize`.
kustomize:
  # kustomize.image maps to --kustomize-image of argocd-app-create.
  image:
# helm instructs kanvas to deploy the app using `helm`.
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
argocd:
  # argocd.repo maps to --repo of argocd-app-create.
  repo: github.com/myorg/myrepo.git
  # argocd.path maps to --path of argocd-app-create.
  # Note: In case you had kubernetes.dir along with argocd.path,
  # kanvas automatically git-push the content of dir to $argocd_repo/$argocd_path.
  # To opt-out of it, set `push: false`.
  path: path/to/dir/in/repo
  # --dir-recurse
  dirRecurse: true
  # --dest-namespace
  namespace: default
  # serverFrom maps to --dest-server where the flag value is take from the output of another kanvas component
  serverFrom: component_name.k8s_endpoint
  # Note that the config management plugin definition in the configmap
  # and the --config-management-plugin flag passed to argocd-app-create # command is auto-generated.
```

`kargo` has three commands, `kargo plan`, `kargo apply`, and `kargo export`.

`plan` outputs the diff between the current state and the desired state of your application deployment, so that you can review changes before they are applied.

`apply` runs the deployment`

`export` is used to generate the worfklow definitions for the CI/CD system of your choice, so that the system can run `kargo plan` on each commit or pull request sync, and `apply` on each merge or commit to the main branch.
