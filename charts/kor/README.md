# kor

![Version: 0.2.2](https://img.shields.io/badge/Version-0.2.2-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.6.2](https://img.shields.io/badge/AppVersion-0.6.2-informational?style=flat-square)

A Kubernetes Helm Chart to discover orphaned resources using kor

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| yonahd |  | <https://github.com/yonahd/kor> |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| additionalLabels | object | `{}` | Custom labels to add into metadata |
| cronJob.args[0] | string | `"all"` |  |
| cronJob.command[0] | string | `"kor"` |  |
| cronJob.enabled | bool | `false` |  |
| cronJob.failedJobsHistoryLimit | int | `2` |  |
| cronJob.image.repository | string | `"yonahdissen/kor"` |  |
| cronJob.image.tag | string | `"latest"` |  |
| cronJob.name | string | `"kor"` |  |
| cronJob.namespaced | string | `nil` | Set true/false to explicitly return namespaced/non-namespaced resources |
| cronJob.restartPolicy | string | `"OnFailure"` |  |
| cronJob.schedule | string | `"0 1 * * 1"` |  |
| cronJob.slackAuthToken | string | `""` |  |
| cronJob.slackChannel | string | `""` |  |
| cronJob.slackWebhookUrl | string | `""` |  |
| cronJob.successfulJobsHistoryLimit | int | `3` |  |
| prometheusExporter.args[0] | string | `"exporter"` |  |
| prometheusExporter.command[0] | string | `"kor"` |  |
| prometheusExporter.deployment.affinity | object | `{}` |  |
| prometheusExporter.deployment.image.repository | string | `"yonahdissen/kor"` |  |
| prometheusExporter.deployment.image.tag | string | `"latest"` |  |
| prometheusExporter.deployment.imagePullPolicy | string | `"Always"` |  |
| prometheusExporter.deployment.imagePullSecrets | list | `[]` |  |
| prometheusExporter.deployment.nodeSelector | object | `{}` |  |
| prometheusExporter.deployment.podSecurityContext | object | `{}` |  |
| prometheusExporter.deployment.replicaCount | int | `1` |  |
| prometheusExporter.deployment.resources | object | `{}` |  |
| prometheusExporter.deployment.restartPolicy | string | `"Always"` |  |
| prometheusExporter.deployment.securityContext | object | `{}` |  |
| prometheusExporter.deployment.tolerations | list | `[]` |  |
| prometheusExporter.enabled | bool | `true` |  |
| prometheusExporter.exporterInterval | string | `""` |  |
| prometheusExporter.name | string | `"kor-exporter"` |  |
| prometheusExporter.namespaced | string | `nil` | Set true/false to explicitly return namespaced/non-namespaced resources |
| prometheusExporter.service.port | int | `8080` |  |
| prometheusExporter.service.type | string | `"ClusterIP"` |  |
| prometheusExporter.serviceMonitor.enabled | bool | `true` |  |
| prometheusExporter.serviceMonitor.interval | string | `"30s"` | Set how frequently Prometheus should scrape |
| prometheusExporter.serviceMonitor.labels | object | `{}` | Service monitor labels |
| prometheusExporter.serviceMonitor.metricRelabelings | list | `[]` |  |
| prometheusExporter.serviceMonitor.namespace | string | `""` | Set the namespace the ServiceMonitor should be deployed, if empty namespace will be `.Release.Namespace` |
| prometheusExporter.serviceMonitor.relabelings | list | `[]` |  |
| prometheusExporter.serviceMonitor.targetLabels | list | `[]` | Set of labels to transfer on the Kubernetes Service onto the target. |
| prometheusExporter.serviceMonitor.telemetryPath | string | `"/metrics"` |  |
| prometheusExporter.serviceMonitor.timeout | string | `"10s"` | Set timeout for scrape |
| serviceAccount.annotations | object | `{}` | Annotations to add to the service account |
| serviceAccount.create | bool | `true` | Specifies whether a service account should be created |
| serviceAccount.name | string | `""` | If not set and create is true, a name is generated using the fullname template |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.14.2](https://github.com/norwoodj/helm-docs/releases/v1.14.2)
