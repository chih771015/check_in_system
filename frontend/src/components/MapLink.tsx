import { EnvironmentOutlined } from '@ant-design/icons';
import { Tooltip } from 'antd';

interface MapLinkProps {
  latitude: number;
  longitude: number;
  address?: string;
  /** 是否顯示地址文字，預設 true */
  showAddress?: boolean;
}

/**
 * 顯示地址文字，並在右側附上可點擊的地圖圖示。
 * - iOS / macOS Safari → Apple Maps
 * - 其他              → Google Maps
 * 若 lat/lng 為 0（未取得定位）則不顯示連結。
 */
export default function MapLink({ latitude, longitude, address, showAddress = true }: MapLinkProps) {
  const hasCoords = latitude !== 0 || longitude !== 0;

  const openMap = () => {
    if (!hasCoords) return;
    const isApple = /iPad|iPhone|iPod|Macintosh/.test(navigator.userAgent) &&
      'ontouchend' in document;
    const url = isApple
      ? `https://maps.apple.com/?q=${latitude},${longitude}`
      : `https://www.google.com/maps?q=${latitude},${longitude}`;
    window.open(url, '_blank', 'noopener,noreferrer');
  };

  const coordsText = hasCoords ? `${latitude.toFixed(6)}, ${longitude.toFixed(6)}` : null;

  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4, flexWrap: 'wrap' }}>
      {showAddress && (
        <span>{address || coordsText || '—'}</span>
      )}
      {hasCoords && (
        <Tooltip title={`${coordsText}　點擊開啟地圖`}>
          <EnvironmentOutlined
            onClick={openMap}
            style={{ color: '#1677ff', cursor: 'pointer', fontSize: 15, flexShrink: 0 }}
          />
        </Tooltip>
      )}
    </span>
  );
}
