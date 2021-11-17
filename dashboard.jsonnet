local grafana = import 'grafonnet-lib/grafonnet/grafana.libsonnet';

local panel(global, metric) = grafana.graphPanel.new(
  metric.name,
  format='s',
  datasource=global.datasource,
  span=2,
).addTarget(
  grafana.prometheus.target(
    metric.expr,
  )
);

local panels(global, metrics) = [panel(global, m) for m in metrics];

local globalDefaults = {
  datasource: '',
};

function(global=globalDefaults, metrics=[])
  grafana.dashboard.new(
    'App Name',
    description='Autogenerated by simple-grafonnet',
    editable=true
  ).addPanels(
    panels(global, metrics)
  )