import { useEffect, useState } from 'react';
import api from '../api/client';
import { DevicePhoneMobileIcon, NoSymbolIcon } from '@heroicons/react/24/outline';

interface Device {
  id: string;
  label: string;
  last_seen_at: string;
  fingerprint: string;
  approved: boolean;
}

const Devices = () => {
  const [devices, setDevices] = useState<Device[]>([]);

  useEffect(() => {
    (async () => {
      try {
        const { data } = await api.get('/devices');
        setDevices(data);
      } catch (err) {
        console.error(err);
      }
    })();
  }, []);

  const revoke = async (id: string) => {
    await api.delete(`/devices/${id}`);
    setDevices((prev) => prev.filter((device) => device.id !== id));
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="section-title">设备指纹</h2>
      </div>
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {devices.map((device) => (
          <div key={device.id} className="flex flex-col rounded-3xl border border-white bg-white/90 p-5 shadow-glow">
            <div className="flex items-center gap-3">
              <DevicePhoneMobileIcon className="h-6 w-6 text-neon-500" />
              <div>
                <p className="text-sm font-medium text-night-100">{device.label}</p>
                <p className="text-xs text-night-300">
                  最近活跃：{new Date(device.last_seen_at).toLocaleString()}
                </p>
              </div>
            </div>
            <p className="mt-4 break-all rounded-2xl bg-ink-50 p-3 text-[11px] leading-relaxed text-night-300">
              {device.fingerprint}
            </p>
            <div className="mt-4 flex items-center justify-between text-xs text-night-300">
              <span className={device.approved ? 'text-neon-500' : 'text-red-400'}>
                {device.approved ? '已授权' : '待审核'}
              </span>
              <button
                onClick={() => revoke(device.id)}
                className="inline-flex items-center gap-1 rounded-full border border-red-200 px-3 py-1 text-red-400 transition-colors hover:border-red-300 hover:text-red-300"
              >
                <NoSymbolIcon className="h-4 w-4" />
                撤销
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default Devices;
