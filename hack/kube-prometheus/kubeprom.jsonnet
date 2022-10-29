local kp =
  (import 'kube-prometheus/main.libsonnet') +
  (import 'kube-prometheus/addons/all-namespaces.libsonnet') +
  (import 'kube-prometheus/addons/custom-metrics.libsonnet') +
  {
    values+:: {
      common+: {
        namespace: 'monitoring',
      },
      prometheus+: {
        name: 'system',
        namespaces: [],
        replicas: 1,
        resources: {
          limits: {
            cpu: '50m',
            memory: '128Mi',
          },
          requests: {
            cpu: '10m',
            memory: '64Mi',
          },
        },
      },
      prometheusAdapter+: {
        namespace: 'monitoring',
        replicas: 1,
        resources: {
          limits: {
            cpu: '50m',
            memory: '128Mi',
          },
          requests: {
            cpu: '10m',
            memory: '64Mi',
          },
        },
      },
    },
  };

{ 'setup/0namespace-namespace': kp.kubePrometheus.namespace } +
{ ['setup/prometheus-operator-' + name]: kp.prometheusOperator[name] for name in std.filter((function(name) name != 'serviceMonitor' && name != 'prometheusRule'), std.objectFields(kp.prometheusOperator)) } +
{ 'prometheus-operator-serviceMonitor': kp.prometheusOperator.serviceMonitor } +
{ 'prometheus-operator-prometheusRule': kp.prometheusOperator.prometheusRule } +
{ ['prometheus-' + name]: kp.prometheus[name] for name in std.objectFields(kp.prometheus) } +
{ ['prometheus-adapter-' + name]: kp.prometheusAdapter[name] for name in std.objectFields(kp.prometheusAdapter) }
