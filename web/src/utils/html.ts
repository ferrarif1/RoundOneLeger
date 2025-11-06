const readBlobAsDataUrl = (blob: Blob): Promise<string> =>
  new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onloadend = () => {
      const { result } = reader;
      if (typeof result === 'string') {
        resolve(result);
      } else {
        reject(new Error('无法解析为数据 URL'));
      }
    };
    reader.onerror = () => {
      reject(reader.error ?? new Error('读取资源失败'));
    };
    reader.readAsDataURL(blob);
  });

const convertAttributeToDataUrl = async (
  element: Element,
  attribute: string,
  cache: Map<string, string>
) => {
  const src = element.getAttribute(attribute);
  if (!src || !src.startsWith('blob:')) {
    return;
  }
  if (cache.has(src)) {
    element.setAttribute(attribute, cache.get(src) ?? '');
    return;
  }
  try {
    const response = await fetch(src);
    if (!response.ok) {
      throw new Error(`请求失败: ${response.status}`);
    }
    const blob = await response.blob();
    const dataUrl = await readBlobAsDataUrl(blob);
    cache.set(src, dataUrl);
    element.setAttribute(attribute, dataUrl);
  } catch (error) {
    // eslint-disable-next-line no-console
    console.warn('无法内联 blob 资源', attribute, src, error);
  }
};

/**
 * Replaces any blob: URLs in media elements with embedded data URLs so the
 * content persists across exports/imports.
 */
export const inlineBlobSources = async (html: string): Promise<string> => {
  if (typeof document === 'undefined' || !html || !html.includes('blob:')) {
    return html;
  }

  const wrapper = document.createElement('div');
  wrapper.innerHTML = html;
  const cache = new Map<string, string>();

  const elements = Array.from(
    wrapper.querySelectorAll('img[src^="blob:"],audio[src^="blob:"],video[src^="blob:"],source[src^="blob:"]')
  );
  const posterElements = Array.from(wrapper.querySelectorAll('video[poster^="blob:"]'));

  await Promise.all([
    ...elements.map((element) => convertAttributeToDataUrl(element, 'src', cache)),
    ...posterElements.map((element) => convertAttributeToDataUrl(element, 'poster', cache))
  ]);

  return wrapper.innerHTML;
};

