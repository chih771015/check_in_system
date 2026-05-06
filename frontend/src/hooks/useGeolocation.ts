import { useCallback, useEffect, useState } from 'react';

export type GeoState = 'idle' | 'requesting' | 'success' | 'denied' | 'unavailable' | 'timeout';

/** Nominatim reverse geocoding（與後端使用相同服務）。失敗時回傳 null，不阻斷打卡。 */
async function reverseGeocode(lat: number, lng: number): Promise<string | null> {
  try {
    const url = `https://nominatim.openstreetmap.org/reverse?format=json&lat=${lat}&lon=${lng}&accept-language=zh-TW`;
    const res = await fetch(url, {
      headers: { 'User-Agent': 'translator-checkin/1.0' },
    });
    if (!res.ok) return null;
    const data = await res.json() as { display_name?: string };
    return data.display_name ?? null;
  } catch {
    return null;
  }
}

export interface GeolocationResult {
  state: GeoState;
  latitude: number | null;
  longitude: number | null;
  address: string;
  request: () => void;
}

/**
 * useGeolocation — 定位 hook
 *
 * 狀態機：
 *   idle        → 尚未請求（首次進入或瀏覽器不支援 Permissions API）
 *   requesting  → 等待瀏覽器回應
 *   success     → 取得座標
 *   denied      → 使用者或系統拒絕
 *   unavailable → 裝置無法定位（GPS 訊號差等）
 *   timeout     → 超過 10 秒未回應
 *
 * 流程：
 *   1. mount 時查詢 navigator.permissions（若支援）
 *      - granted  → 直接自動請求
 *      - denied   → 進入 denied 狀態，不再彈對話框
 *      - prompt   → 進入 idle，等使用者主動按授權按鈕（permission priming）
 *   2. 若瀏覽器不支援 Permissions API → 進入 idle，讓使用者手動按
 */
export function useGeolocation(): GeolocationResult {
  const [state, setState] = useState<GeoState>('idle');
  const [latitude, setLatitude] = useState<number | null>(null);
  const [longitude, setLongitude] = useState<number | null>(null);
  const [address, setAddress] = useState('');

  const request = useCallback(() => {
    if (!navigator.geolocation) {
      setState('unavailable');
      return;
    }
    setState('requesting');
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        const lat = pos.coords.latitude;
        const lng = pos.coords.longitude;
        setLatitude(lat);
        setLongitude(lng);
        // 先顯示座標，Nominatim 回來後換成文字地址
        setAddress(`${lat.toFixed(6)}, ${lng.toFixed(6)}`);
        setState('success');
        reverseGeocode(lat, lng).then((name) => {
          if (name) setAddress(name);
        });
      },
      (err) => {
        if (err.code === GeolocationPositionError.PERMISSION_DENIED) {
          setState('denied');
        } else if (err.code === GeolocationPositionError.TIMEOUT) {
          setState('timeout');
        } else {
          setState('unavailable');
        }
      },
      { enableHighAccuracy: true, timeout: 10000 },
    );
  }, []);

  useEffect(() => {
    if (!navigator.geolocation) {
      setState('unavailable');
      return;
    }
    // 若瀏覽器不支援 Permissions API，讓使用者自行點按鈕
    if (!navigator.permissions) return;

    navigator.permissions
      .query({ name: 'geolocation' })
      .then((result) => {
        if (result.state === 'granted') {
          // 已授權 → 直接取得，不打擾使用者
          request();
        } else if (result.state === 'denied') {
          // 已拒絕 → 不再彈對話框，直接告知
          setState('denied');
        }
        // 'prompt' → 保持 idle，等使用者主動授權
      })
      .catch(() => {
        // 查詢失敗（部分瀏覽器限制）→ 保持 idle
      });
  }, [request]);

  return { state, latitude, longitude, address, request };
}
