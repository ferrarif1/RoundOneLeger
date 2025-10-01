const textEncoder = new TextEncoder();

const navigatorInfo = () => {
  const { userAgent, language, languages, platform } = window.navigator;
  return [userAgent, language, languages?.join('|'), platform].join('::');
};

const screenInfo = () => {
  const { colorDepth, height, width, pixelDepth } = window.screen;
  return [colorDepth, height, width, pixelDepth].join('x');
};

const timezoneInfo = () => Intl.DateTimeFormat().resolvedOptions().timeZone || 'unknown';

export const collectFingerprint = () => {
  const canvas = document.createElement('canvas');
  const context = canvas.getContext('2d');
  context?.fillText('ledger-eidos-theme', 0, 0);
  const canvasHash = canvas.toDataURL();

  return [navigatorInfo(), screenInfo(), timezoneInfo(), canvasHash].join('::');
};

export const fingerprintHash = async () => {
  const fingerprint = collectFingerprint();
  const key = await crypto.subtle.importKey(
    'raw',
    textEncoder.encode(import.meta.env.VITE_FINGERPRINT_SECRET || 'ledger-default-secret'),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign']
  );

  const signature = await crypto.subtle.sign('HMAC', key, textEncoder.encode(fingerprint));
  return btoa(String.fromCharCode(...new Uint8Array(signature)));
};
