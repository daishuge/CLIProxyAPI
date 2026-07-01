import { useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { INLINE_LOGO_JPEG } from '@/assets/logoInline';
import { useSplashTitleFit } from '@/hooks/useSplashTitleFit';
import './SplashScreen.scss';

interface SplashScreenProps {
  onFinish: () => void;
  fadeOut?: boolean;
}

const FADE_OUT_DURATION = 400;

export function SplashScreen({ onFinish, fadeOut = false }: SplashScreenProps) {
  const { t } = useTranslation();
  const splashContentRef = useRef<HTMLDivElement>(null);
  const splashTitleRef = useRef<HTMLHeadingElement>(null);
  const splashTitle = t('splash.title');

  useSplashTitleFit(splashContentRef, splashTitleRef, splashTitle);

  useEffect(() => {
    if (!fadeOut) return;
    const finishTimer = setTimeout(() => {
      onFinish();
    }, FADE_OUT_DURATION);

    return () => {
      clearTimeout(finishTimer);
    };
  }, [fadeOut, onFinish]);

  return (
    <div className={`splash-screen ${fadeOut ? 'fade-out' : ''}`}>
      <div ref={splashContentRef} className="splash-content">
        <img src={INLINE_LOGO_JPEG} alt="Playful Proxy API Panel" className="splash-logo" />
        <h1 ref={splashTitleRef} className="splash-title" aria-label={splashTitle}>
          <span>{t('splash.title_line_1')}</span>
          <span>{t('splash.title_line_2')}</span>
        </h1>
        <p className="splash-subtitle">{t('splash.subtitle')}</p>
        <div className="splash-loader">
          <div className="splash-loader-bar" />
        </div>
      </div>
    </div>
  );
}
