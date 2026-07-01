import { useLayoutEffect, type RefObject } from 'react';

const DEFAULT_MIN_FONT_SIZE = 32;
const DEFAULT_MAX_FONT_SIZE = 118;
const DEFAULT_VARIABLE_NAME = '--splash-title-size';

type SplashTitleFitOptions = {
  minFontSize?: number;
  maxFontSize?: number;
  variableName?: string;
};

export function useSplashTitleFit(
  containerRef: RefObject<HTMLElement | null>,
  titleRef: RefObject<HTMLElement | null>,
  dependencyKey: string,
  {
    minFontSize = DEFAULT_MIN_FONT_SIZE,
    maxFontSize = DEFAULT_MAX_FONT_SIZE,
    variableName = DEFAULT_VARIABLE_NAME,
  }: SplashTitleFitOptions = {}
) {
  useLayoutEffect(() => {
    const container = containerRef.current;
    const title = titleRef.current;
    if (!container || !title) return;

    let frameId = 0;
    const fitParent = container.parentElement;

    const fits = (fontSize: number) => {
      title.style.setProperty(variableName, `${fontSize}px`);

      const widthLimit = container.clientWidth;
      const heightLimit = fitParent?.clientHeight ?? Number.POSITIVE_INFINITY;

      return title.scrollWidth <= widthLimit + 1 && container.scrollHeight <= heightLimit + 1;
    };

    const update = () => {
      frameId = 0;

      let low = minFontSize;
      let high = maxFontSize;

      for (let i = 0; i < 8; i += 1) {
        const mid = (low + high) / 2;
        if (fits(mid)) {
          low = mid;
        } else {
          high = mid;
        }
      }

      title.style.setProperty(variableName, `${Math.floor(low)}px`);
    };

    const schedule = () => {
      if (frameId) {
        cancelAnimationFrame(frameId);
      }
      frameId = requestAnimationFrame(update);
    };

    schedule();

    const observer = typeof ResizeObserver === 'undefined' ? null : new ResizeObserver(schedule);
    observer?.observe(container);
    if (fitParent) {
      observer?.observe(fitParent);
    }

    document.fonts?.ready.then(schedule).catch(() => undefined);
    window.addEventListener('resize', schedule);

    return () => {
      if (frameId) {
        cancelAnimationFrame(frameId);
      }
      observer?.disconnect();
      window.removeEventListener('resize', schedule);
    };
  }, [containerRef, dependencyKey, maxFontSize, minFontSize, titleRef, variableName]);
}
