local grafana = import 'grafonnet-lib/grafonnet/grafana.libsonnet';

local newPanel(global, metric, x, width) =
  local p = grafana.graphPanel.new(
    metric.title,
    description=metric.name,
    format=metric.format,
    datasource=global.datasource,
    legend_alignAsTable=true,
    legend_avg=true,
    legend_min=true,
    legend_max=true,
    legend_current=true,
    legend_sortDesc=true,
    legend_values=true,
  ).addTarget(
    grafana.prometheus.target(
      metric.expr,
    )
  );
  p {
    gridPos: {
      w: width,
      h: 10,
      x: x,
      y: 0,
    },
  };

local globalDefaults = {
  title: 'App Name',
  datasource: '',
};

local addPanelsWithRows(dash, metric) =
  local pos = { x: 0, y: 0, w: 12, h: 9 };
  local title = metric.group + ' - ' + metric.subgroup;
  local isNewRow = dash.row.title != title;
  local panel = newPanel(dash.global, metric, 0, 12);

  local row = if isNewRow
  then grafana.row.new(collapse=true, title=title).addPanel(panel, pos)
  else dash.row.addPanel(panel, pos);

  local dashboard = if isNewRow && dash.row.title != ''
  then dash.dashboard.addRow(dash.row)
  else dash.dashboard;

  dash {
    dashboard: dashboard,
    row: row,
  };


function(global=globalDefaults, metrics=[
  { expr: 'expr1', title: 'title1', format: 'simple', group: 'g1', subgroup: 'g' },
  { expr: 'expr2', title: 'title2', format: 'simple', group: 'g2', subgroup: 'g' },
])
  local dash = {
    global: global,
    dashboard: grafana.dashboard.new(
      global.title,
      description='Autogenerated by simple-grafonnet',
      editable=true,
      time_from='now-30m',
    ),
    row: { title: '' },
  };

  local foldl = std.foldl(addPanelsWithRows, metrics, dash);
  local withLastRow = foldl { dashboard: foldl.dashboard.addRow(foldl.row) };
  withLastRow.dashboard
