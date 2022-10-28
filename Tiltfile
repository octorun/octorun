# -*- mode: Python -*-
# pyright: reportUndefinedVariable=false

os.putenv('PATH', './bin' + ':' + os.getenv('PATH'))


def fixup_run_as_nonroot(yaml):
    yaml_str = str(yaml)
    return blob(yaml_str.replace("runAsNonRoot: true", "runAsNonRoot: false"))


load('ext://cert_manager', 'deploy_cert_manager')
deploy_cert_manager()

local("make kustomize", quiet=True)

manager_deps = ["api", "controllers",
                "webhooks", "go.mod", "go.sum", "main.go"]
manager_ignore = ['*/*/zz_generated.deepcopy.go']

local_resource("octorun-manifest",
               cmd='make manifests',
               trigger_mode=TRIGGER_MODE_AUTO,
               deps=manager_deps,
               ignore=manager_ignore,
               resource_deps=[],
               labels=[])

manager_deps.append("hooks")
manager_deps.append("metrics")
manager_deps.append("pkg")
manager_deps.append("util")
local_resource("octorun-manager-binary",
               cmd='make manager',
               env={
                   "GOOS": "linux",
                   "GOARCH": "amd64"
               },
               trigger_mode=TRIGGER_MODE_AUTO,
               deps=manager_deps,
               ignore=manager_ignore,
               resource_deps=[],
               labels=[])

dockerfile = """
FROM golang:1.19 as tilt-helper
# Support live reloading with Tilt
RUN wget --output-document /restart.sh --quiet https://raw.githubusercontent.com/tilt-dev/rerun-process-wrapper/master/restart.sh  && \
    wget --output-document /start.sh --quiet https://raw.githubusercontent.com/tilt-dev/rerun-process-wrapper/master/start.sh && \
    chmod +x /start.sh && chmod +x /restart.sh

FROM gcr.io/distroless/base:debug as manager
WORKDIR /
COPY --from=tilt-helper /start.sh .
COPY --from=tilt-helper /restart.sh .
COPY manager .
"""

docker_build(
    ref="ghcr.io/octorun/manager",
    context="bin/",
    dockerfile_contents=dockerfile,
    target="manager",
    entrypoint=["sh", "/start.sh", "/manager"],
    only="manager",
    live_update=[
        sync("bin/manager", "/manager"),
        run("sh /restart.sh"),
    ])


k8s_yaml(fixup_run_as_nonroot(kustomize("config/default")))
k8s_resource('octorun-manager', resource_deps=['octorun-manifest', 'octorun-manager-binary'], objects=[
    "octorun-system:namespace",
    "octorun-serving-cert:certificate",
    "octorun-selfsigned-issuer:issuer",
    "octorun-metrics-reader:clusterrole",
    "octorun-manager:serviceaccount",
    "octorun-manager-secret:secret",
    "octorun-manager-config:configmap",
    "octorun-manager-role:clusterrole",
    "octorun-manager-rolebinding:clusterrolebinding",
    "octorun-manager-metrics:servicemonitor",
    "octorun-runner-editor-role:clusterrole",
    "octorun-runner-viewer-role:clusterrole",
    "octorun-runnerset-editor-role:clusterrole",
    "octorun-runnerset-viewer-role:clusterrole",
    "octorun-proxy-role:clusterrole",
    "octorun-proxy-rolebinding:clusterrolebinding",
    "octorun-leader-election-role:role",
    "octorun-leader-election-rolebinding:rolebinding",
    "runners.octorun.github.io:customresourcedefinition",
    "runnersets.octorun.github.io:customresourcedefinition",
    "octorun-mutating-webhook-configuration:mutatingwebhookconfiguration",
    "octorun-validating-webhook-configuration:validatingwebhookconfiguration",
])
