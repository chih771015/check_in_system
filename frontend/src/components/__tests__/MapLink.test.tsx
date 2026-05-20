import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import MapLink from '../MapLink';

describe('MapLink', () => {
  let originalOpen: typeof window.open;

  beforeEach(() => {
    originalOpen = window.open;
    window.open = vi.fn();
  });

  afterEach(() => {
    cleanup();
    window.open = originalOpen;
  });

  it('renders address text + icon when coordinates are non-zero', () => {
    render(<MapLink latitude={25.0} longitude={121.5} address="Taipei" />);
    expect(screen.getByText('Taipei')).toBeInTheDocument();
  });

  it('renders coordinates fallback when address is empty but coords present', () => {
    render(<MapLink latitude={25.123456} longitude={121.654321} />);
    expect(screen.getByText('25.123456, 121.654321')).toBeInTheDocument();
  });

  it('renders em-dash when neither address nor coordinates are available', () => {
    render(<MapLink latitude={0} longitude={0} />);
    expect(screen.getByText('—')).toBeInTheDocument();
  });

  it('does not render the map icon when lat=lng=0', () => {
    const { container } = render(<MapLink latitude={0} longitude={0} address="No GPS" />);
    // EnvironmentOutlined renders an <svg>; with no coords there should be none.
    expect(container.querySelector('svg')).toBeNull();
  });

  it('hides address when showAddress is false', () => {
    render(<MapLink latitude={25.0} longitude={121.5} address="Taipei" showAddress={false} />);
    expect(screen.queryByText('Taipei')).toBeNull();
  });

  it('opens Google Maps URL on icon click in a non-Apple environment', async () => {
    // happy-dom defaults to a non-Mac userAgent and no ontouchend
    render(<MapLink latitude={25.0} longitude={121.5} address="x" />);
    const user = userEvent.setup();

    const icon = document.querySelector('svg');
    expect(icon).not.toBeNull();
    await user.click(icon!.parentElement!);

    expect(window.open).toHaveBeenCalledTimes(1);
    const url = (window.open as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(url).toContain('google.com/maps');
    expect(url).toContain('25,121.5');
  });
});
