import { describe, expect, test } from 'bun:test';
import { createElement, useState } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { parse as parseYaml } from 'yaml';
import { useVisualConfig } from '../src/hooks/useVisualConfig';

describe('visual config PPAP extensions', () => {
  test('round-trips preset prompt and concurrency without touching custom YAML', () => {
    function Harness() {
      const visualConfig = useVisualConfig();
      const [phase, setPhase] = useState(0);

      if (phase === 0) {
        visualConfig.loadVisualValuesFromYaml(
          [
            'preset-prompt:',
            '  enabled: false',
            '  prompt: old prompt',
            '  max-bytes: 1024',
            'upstream-concurrency:',
            '  default: 2',
            '  queue-timeout-seconds: 10',
            'api-key-controls:',
            '  - api-key: keep-me',
            '',
          ].join('\n')
        );
        setPhase(1);
      } else if (phase === 1) {
        visualConfig.setVisualValues({
          ppapPresetPromptEnabled: true,
          ppapPresetPromptText: 'new prompt',
          ppapPresetPromptMaxBytes: '32768',
          ppapUpstreamConcurrencyDefault: '6',
          ppapUpstreamConcurrencyQueueTimeoutSeconds: '45',
        });
        setPhase(2);
      } else {
        return createElement(
          'pre',
          null,
          visualConfig.applyVisualChangesToYaml(
            [
              'preset-prompt:',
              '  enabled: false',
              '  prompt: old prompt',
              '  max-bytes: 1024',
              'upstream-concurrency:',
              '  default: 2',
              '  queue-timeout-seconds: 10',
              'api-key-controls:',
              '  - api-key: keep-me',
              '',
            ].join('\n')
          )
        );
      }

      return null;
    }

    const markup = renderToStaticMarkup(createElement(Harness));
    const result = parseYaml(markup.slice('<pre>'.length, -'</pre>'.length));

    expect(result['preset-prompt']).toEqual({
      enabled: true,
      prompt: 'new prompt',
      'max-bytes': 32768,
    });
    expect(result['upstream-concurrency']).toEqual({
      default: 6,
      'queue-timeout-seconds': 45,
    });
    expect(result['api-key-controls']).toEqual([{ 'api-key': 'keep-me' }]);
  });
});
