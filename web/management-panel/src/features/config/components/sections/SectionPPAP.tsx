import { useTranslation } from 'react-i18next';
import { Collapsible } from '@/components/ui/Collapsible';
import { Input } from '@/components/ui/Input';
import { CONFIG_TAB_ICONS, SECTION_INDEX_LABELS } from '../../constants';
import type { ConfigSectionProps } from '../../types';
import { SectionCard } from '../SectionCard';
import {
  Divider,
  FieldAnchor,
  FieldGrid,
  FieldHint,
  FieldShell,
  FieldStack,
  ToggleRow,
} from '../fields/FieldPrimitives';

const Icon = CONFIG_TAB_ICONS.ppap;

export function SectionPPAP({ values, disabled, animateIn, onChange }: ConfigSectionProps) {
  const { t } = useTranslation();

  return (
    <SectionCard
      indexLabel={SECTION_INDEX_LABELS.ppap}
      icon={<Icon size={16} />}
      title={t('config_management.visual.sections.ppap.title')}
      description={t('config_management.visual.sections.ppap.description')}
      animateIn={animateIn}
    >
      <FieldStack>
        <Collapsible
          label={t('config_management.visual.sections.ppap.preset_prompt_group')}
          hint={t('config_management.visual.sections.ppap.preset_prompt_group_desc')}
          defaultOpen
        >
          <FieldStack>
            <FieldAnchor fieldId="ppapPresetPromptEnabled">
              <ToggleRow
                title={t('config_management.visual.sections.ppap.preset_prompt_enabled')}
                description={t(
                  'config_management.visual.sections.ppap.preset_prompt_enabled_hint'
                )}
                checked={values.ppapPresetPromptEnabled}
                disabled={disabled}
                onChange={(ppapPresetPromptEnabled) => onChange({ ppapPresetPromptEnabled })}
              />
            </FieldAnchor>
            <FieldAnchor fieldId="ppapPresetPromptText">
              <FieldShell
                label={t('config_management.visual.sections.ppap.preset_prompt_text')}
                hint={t('config_management.visual.sections.ppap.preset_prompt_text_hint')}
              >
                <textarea
                  className="input"
                  rows={6}
                  value={values.ppapPresetPromptText}
                  placeholder={t(
                    'config_management.visual.sections.ppap.preset_prompt_text_placeholder'
                  )}
                  onChange={(event) => onChange({ ppapPresetPromptText: event.target.value })}
                  disabled={disabled}
                />
              </FieldShell>
            </FieldAnchor>
            <FieldGrid>
              <FieldAnchor fieldId="ppapPresetPromptMaxBytes">
                <Input
                  label={t('config_management.visual.sections.ppap.preset_prompt_max_bytes')}
                  type="number"
                  min="0"
                  max="262144"
                  placeholder="32768"
                  value={values.ppapPresetPromptMaxBytes}
                  onChange={(event) =>
                    onChange({ ppapPresetPromptMaxBytes: event.target.value })
                  }
                  disabled={disabled}
                  hint={t(
                    'config_management.visual.sections.ppap.preset_prompt_max_bytes_hint'
                  )}
                />
              </FieldAnchor>
            </FieldGrid>
          </FieldStack>
        </Collapsible>

        <Divider />

        <Collapsible
          label={t('config_management.visual.sections.ppap.upstream_concurrency_group')}
          hint={t('config_management.visual.sections.ppap.upstream_concurrency_group_desc')}
          defaultOpen
        >
          <FieldGrid>
            <FieldAnchor fieldId="ppapUpstreamConcurrencyDefault">
              <Input
                label={t('config_management.visual.sections.ppap.upstream_concurrency_default')}
                type="number"
                min="0"
                placeholder="0"
                value={values.ppapUpstreamConcurrencyDefault}
                onChange={(event) =>
                  onChange({ ppapUpstreamConcurrencyDefault: event.target.value })
                }
                disabled={disabled}
                hint={t(
                  'config_management.visual.sections.ppap.upstream_concurrency_default_hint'
                )}
              />
            </FieldAnchor>
            <FieldAnchor fieldId="ppapUpstreamConcurrencyQueueTimeoutSeconds">
              <Input
                label={t(
                  'config_management.visual.sections.ppap.upstream_concurrency_queue_timeout'
                )}
                type="number"
                min="0"
                placeholder="30"
                value={values.ppapUpstreamConcurrencyQueueTimeoutSeconds}
                onChange={(event) =>
                  onChange({
                    ppapUpstreamConcurrencyQueueTimeoutSeconds: event.target.value,
                  })
                }
                disabled={disabled}
                hint={t(
                  'config_management.visual.sections.ppap.upstream_concurrency_queue_timeout_hint'
                )}
              />
            </FieldAnchor>
          </FieldGrid>
        </Collapsible>

        <Divider />

        <Collapsible
          label={t('config_management.visual.sections.ppap.raw_yaml_group')}
          hint={t('config_management.visual.sections.ppap.raw_yaml_group_desc')}
          defaultOpen={false}
        >
          <FieldStack>
            <FieldHint>
              {t('config_management.visual.sections.ppap.raw_yaml_api_key_controls')}
            </FieldHint>
            <FieldHint>
              {t('config_management.visual.sections.ppap.raw_yaml_immersive_translate')}
            </FieldHint>
            <FieldHint>
              {t('config_management.visual.sections.ppap.raw_yaml_log_controls')}
            </FieldHint>
          </FieldStack>
        </Collapsible>
      </FieldStack>
    </SectionCard>
  );
}
