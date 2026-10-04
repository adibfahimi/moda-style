import { render, screen } from '@solidjs/testing-library';
import { describe, expect, it } from 'vitest';
import StatsCard from './StatsCard';

describe('StatsCard', () => {
  it('renders the title, value and icon', () => {
    render(() => <StatsCard title="Total products" value={128} icon="👗" />);

    expect(screen.getByText('Total products')).toBeDefined();
    expect(screen.getByText('128')).toBeDefined();
    expect(screen.getByText('👗')).toBeDefined();
  });

  it('renders the description when one is given', () => {
    render(
      () => <StatsCard title="Orders" value={12} icon="🧾" description="Last 30 days" />,
    );

    expect(screen.getByText('Last 30 days')).toBeDefined();
  });

  it('renders an upward trend in the success colour', () => {
    render(
      () => (
        <StatsCard
          title="Revenue"
          value="€4,200"
          icon="💶"
          trend="up"
          trendValue="12%"
        />
      ),
    );

    const trend = screen.getByText(/12%/);

    expect(trend.className).toContain('text-success');
    expect(trend.textContent).toContain('↗');
  });

  it('renders a downward trend in the error colour', () => {
    render(
      () => (
        <StatsCard
          title="Stock alerts"
          value={3}
          icon="⚠️"
          trend="down"
          trendValue="8%"
        />
      ),
    );

    const trend = screen.getByText(/8%/);

    expect(trend.className).toContain('text-error');
    expect(trend.textContent).toContain('↘');
  });

  it('omits the trend row when no trend value is supplied', () => {
    render(() => <StatsCard title="Reviews" value={9} icon="⭐" trend="up" />);

    expect(screen.queryByText('↗︎')).toBeNull();
  });
});
