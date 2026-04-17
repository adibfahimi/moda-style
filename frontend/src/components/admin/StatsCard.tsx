interface StatsCardProps {
  title: string;
  value: string | number;
  icon: string;
  description?: string;
  trend?: 'up' | 'down';
  trendValue?: string;
  color?: 'primary' | 'secondary' | 'accent' | 'success' | 'warning' | 'error' | 'info';
}

export default function StatsCard(props: StatsCardProps) {
  return (
    <div class="stats shadow">
      <div class="stat">
        <div class="stat-figure text-4xl">{props.icon}</div>
        <div class="stat-title">{props.title}</div>
        <div class="stat-value">{props.value}</div>
        {props.description && (
          <div class="stat-desc">{props.description}</div>
        )}
        {props.trend && props.trendValue && (
          <div class={`stat-desc ${props.trend === 'up' ? 'text-success' : 'text-error'}`}>
            {props.trend === 'up' ? '↗︎' : '↘︎'} {props.trendValue}
          </div>
        )}
      </div>
    </div>
  );
}
